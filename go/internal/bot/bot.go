// Package bot は Discord Gateway への接続、イベント受信、スラッシュコマンドを扱う。
package bot

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/disgoorg/disgo"
	disgobot "github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"

	"github.com/varubogu/discord_message_cleaner_go/internal/config"
	"github.com/varubogu/discord_message_cleaner_go/internal/deleter"
	"github.com/varubogu/discord_message_cleaner_go/internal/discordapi"
	"github.com/varubogu/discord_message_cleaner_go/internal/i18n"
	"github.com/varubogu/discord_message_cleaner_go/internal/registry"
	"github.com/varubogu/discord_message_cleaner_go/internal/store"
	"github.com/varubogu/discord_message_cleaner_go/internal/syncer"
	"github.com/varubogu/discord_message_cleaner_go/internal/tracker"
)

// Deps は Bot が使うコンポーネント。
type Deps struct {
	Config   *config.Config
	Store    *store.Store
	Registry *registry.Registry
	Messages *i18n.Messages
	Logger   *slog.Logger
}

// Bot は Discord Bot 本体。
type Bot struct {
	cfg      *config.Config
	store    *store.Store
	reg      *registry.Registry
	msgs     *i18n.Messages
	log      *slog.Logger
	client   *disgobot.Client
	api      *discordapi.API
	tracker  *tracker.Tracker
	deleter  *deleter.Deleter
	syncer   *syncer.Syncer
	confirms *confirms

	// ctx はバックグラウンド処理（同期など）用。Run の ctx が入る。
	ctx context.Context
}

// New は Bot を作成する。Gateway にはまだ接続しない。
func New(d Deps) (*Bot, error) {
	b := &Bot{
		cfg:   d.Config,
		store: d.Store,
		reg:   d.Registry,
		msgs:  d.Messages,
		log:   d.Logger,
		ctx:   context.Background(),
	}

	client, err := disgo.New(d.Config.DiscordToken,
		disgobot.WithLogger(d.Logger),
		// メッセージ本文は不要なので特権インテント（MESSAGE_CONTENT）は使わない
		disgobot.WithGatewayConfigOpts(
			gateway.WithIntents(gateway.IntentGuilds, gateway.IntentGuildMessages),
		),
		// ロール名の判定と権限計算に必要なものだけキャッシュする（メッセージはキャッシュしない）
		disgobot.WithCacheConfigOpts(
			cache.WithCaches(cache.FlagGuilds, cache.FlagChannels, cache.FlagRoles),
		),
		// コマンド処理などでイベント受信が詰まらないよう、イベントは並行に処理する
		disgobot.WithEventManagerConfigOpts(disgobot.WithAsyncEventsEnabled()),
		disgobot.WithRestClientConfigOpts(
			rest.WithRateLimiterConfigOpts(rest.WithMaxRetries(5)),
		),
		disgobot.WithEventListenerFunc(b.onGuildsReady),
		disgobot.WithEventListenerFunc(b.onMessageCreate),
		disgobot.WithEventListenerFunc(b.onMessageDelete),
		disgobot.WithEventListenerFunc(b.onChannelDelete),
		disgobot.WithEventListenerFunc(b.onThreadDelete),
		disgobot.WithEventListenerFunc(b.onGuildLeave),
		disgobot.WithEventListenerFunc(b.onCommand),
		disgobot.WithEventListenerFunc(b.onComponent),
	)
	if err != nil {
		return nil, fmt.Errorf("create discord client: %w", err)
	}
	b.client = client
	b.api = discordapi.New(client.Rest, d.Config.APIRatePerSecond, d.Config.APIBurst)
	b.tracker = tracker.New(d.Store, d.Registry, d.Config.TrackerFlushInterval, d.Config.TrackerBatchSize, d.Config.TrackerQueueSize, d.Logger)
	b.syncer = syncer.New(d.Store, b.api, d.Registry, d.Config.SyncMaxPages, d.Config.SyncConcurrency, d.Logger)
	b.confirms = newConfirms(client, d.Config.ConfirmTimeout, d.Logger)
	return b, nil
}

// Run は Bot を起動し、ctx が終了するまでブロックする。
func (b *Bot) Run(ctx context.Context) error {
	b.ctx = ctx

	// tracker は deleter より後に止める（deleter が削除完了の記録に使うため）
	trackerCtx, stopTracker := context.WithCancel(context.Background())
	trackerDone := make(chan struct{})
	go func() {
		b.tracker.Run(trackerCtx)
		close(trackerDone)
	}()
	defer func() {
		stopTracker()
		<-trackerDone
	}()

	b.deleter = deleter.New(deleter.Config{
		Interval:              b.cfg.SchedulerInterval,
		ClaimBatchSize:        b.cfg.ClaimBatchSize,
		ClaimLease:            b.cfg.ClaimLease,
		ChannelQueueLimit:     b.cfg.ChannelQueueLimit,
		Workers:               b.cfg.DeleteWorkers,
		MaxAttempts:           b.cfg.MaxDeleteAttempts,
		ChannelFailureLimit:   b.cfg.ChannelFailureLimit,
		ChannelFailureBackoff: b.cfg.ChannelFailureBackoff,
	}, b.store, b.api, removerFunc(func(_ context.Context, id snowflake.ID) {
		b.tracker.Remove(trackerCtx, id)
	}), b.reg, b.log)

	if err := b.registerCommands(); err != nil {
		return err
	}

	if err := b.client.OpenGateway(ctx); err != nil {
		return fmt.Errorf("open gateway: %w", err)
	}
	b.log.Info("bot is online", slog.Int("monitored_channels", len(b.reg.All())))

	deleterDone := make(chan struct{})
	go func() {
		b.deleter.Run(ctx)
		close(deleterDone)
	}()

	<-ctx.Done()
	b.log.Info("shutting down")

	closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	b.client.Close(closeCtx)
	<-deleterDone
	b.syncer.Wait()
	return nil
}

func (b *Bot) registerCommands() error {
	cmds := commandDefinitions()
	var err error
	if b.cfg.DevGuildID != 0 {
		_, err = b.client.Rest.SetGuildCommands(b.client.ApplicationID, b.cfg.DevGuildID, cmds)
	} else {
		_, err = b.client.Rest.SetGlobalCommands(b.client.ApplicationID, cmds)
	}
	if err != nil {
		return fmt.Errorf("register commands: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Gateway イベント
// ---------------------------------------------------------------------------

// onGuildsReady は接続（新規セッション）完了時に、停止中・切断中に投稿されたメッセージを回収する。
// セッション再開（RESUME）時は Discord が取りこぼしたイベントを再送するため呼ばれない。
func (b *Bot) onGuildsReady(e *events.GuildsReady) {
	b.log.Info("guilds ready, start syncing monitored channels", slog.Int("channels", len(b.reg.All())))
	b.syncer.SyncAll(b.ctx)
}

// onMessageCreate はメッセージ投稿を記録する。API リクエストは発生しない。
func (b *Bot) onMessageCreate(e *events.GuildMessageCreate) {
	if !b.reg.IsMonitored(e.ChannelID) {
		return
	}
	if !e.Message.Type.Deleteable() {
		return
	}
	b.tracker.Track(b.ctx, store.Message{ID: e.MessageID, GuildID: e.GuildID, ChannelID: e.ChannelID})
}

// onMessageDelete は他者によって削除されたメッセージを記録から外し、無駄な削除リクエストを防ぐ。
// 一括削除イベント（MESSAGE_DELETE_BULK）も disgo が1件ずつのイベントに展開する。
func (b *Bot) onMessageDelete(e *events.GuildMessageDelete) {
	if !b.reg.IsMonitored(e.ChannelID) {
		return
	}
	b.tracker.Remove(b.ctx, e.MessageID)
}

func (b *Bot) onChannelDelete(e *events.GuildChannelDelete) {
	b.purgeChannel(e.ChannelID)
}

func (b *Bot) onThreadDelete(e *events.ThreadDelete) {
	b.purgeChannel(e.ThreadID)
}

func (b *Bot) purgeChannel(channelID snowflake.ID) {
	if err := b.store.PurgeChannel(b.ctx, channelID); err != nil {
		b.log.Error("failed to purge deleted channel", slog.Uint64("channel_id", uint64(channelID)), slog.Any("err", err))
		return
	}
	if b.reg.IsMonitored(channelID) {
		b.log.Info("monitored channel was deleted, remove settings", slog.Uint64("channel_id", uint64(channelID)))
	}
	b.reg.Delete(channelID)
	if b.deleter != nil {
		b.deleter.DropChannel(channelID)
	}
}

func (b *Bot) onGuildLeave(e *events.GuildLeave) {
	b.log.Info("left guild, remove settings", slog.Uint64("guild_id", uint64(e.GuildID)))
	for _, ch := range b.reg.Guild(e.GuildID) {
		if b.deleter != nil {
			b.deleter.DropChannel(ch.ChannelID)
		}
	}
	b.reg.DeleteGuild(e.GuildID)
	if err := b.store.PurgeGuild(b.ctx, e.GuildID); err != nil {
		b.log.Error("failed to purge guild", slog.Uint64("guild_id", uint64(e.GuildID)), slog.Any("err", err))
	}
}

type removerFunc func(ctx context.Context, id snowflake.ID)

func (f removerFunc) Remove(ctx context.Context, id snowflake.ID) { f(ctx, id) }
