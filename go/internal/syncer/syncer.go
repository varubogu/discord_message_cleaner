// Package syncer はイベントで受け取れなかったメッセージを REST API で補完する。
//
// 通常はメッセージ投稿イベントだけで記録するため API リクエストは発生しない。
// 以下の場合のみ、チャンネルの「同期済み位置（synced_until）」以降を1ページ100件で取得する。
//
//   - 起動時・再接続時（Bot 停止中に投稿されたメッセージの回収）
//   - 監視開始時（BACKFILL_ON_ENABLE=true の場合、既存メッセージの取り込み）
package syncer

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"

	"github.com/varubogu/discord_message_cleaner_go/internal/discordapi"
	"github.com/varubogu/discord_message_cleaner_go/internal/registry"
	"github.com/varubogu/discord_message_cleaner_go/internal/store"
)

const pageSize = 100

// Store は Syncer が使う DB 操作。
type Store interface {
	SyncedUntil(ctx context.Context, channelID snowflake.ID) (snowflake.ID, bool, error)
	TrackMessages(ctx context.Context, msgs []store.Message) (int64, error)
	UpdateSyncedUntil(ctx context.Context, channelID, messageID snowflake.ID) error
	PurgeChannel(ctx context.Context, channelID snowflake.ID) error
}

// Discord は Syncer が使う Discord API。
type Discord interface {
	GetMessagesAfter(ctx context.Context, channelID, after snowflake.ID, limit int) ([]discord.Message, error)
}

// Syncer はチャンネル履歴の同期を行う。
type Syncer struct {
	store    Store
	api      Discord
	reg      *registry.Registry
	maxPages int
	sem      chan struct{}
	log      *slog.Logger

	mu      sync.Mutex
	running map[snowflake.ID]bool
	wg      sync.WaitGroup
}

// New は Syncer を作成する。
func New(s Store, api Discord, reg *registry.Registry, maxPages, concurrency int, log *slog.Logger) *Syncer {
	return &Syncer{
		store:    s,
		api:      api,
		reg:      reg,
		maxPages: maxPages,
		sem:      make(chan struct{}, concurrency),
		log:      log.With(slog.String("component", "syncer")),
		running:  map[snowflake.ID]bool{},
	}
}

// SyncAll は全監視チャンネルの同期を開始する。
func (s *Syncer) SyncAll(ctx context.Context) {
	for _, e := range s.reg.All() {
		s.Start(ctx, e.GuildID, e.ChannelID)
	}
}

// Start はチャンネルの同期をバックグラウンドで開始する。既に同期中なら何もしない。
func (s *Syncer) Start(ctx context.Context, guildID, channelID snowflake.ID) {
	s.mu.Lock()
	if s.running[channelID] {
		s.mu.Unlock()
		return
	}
	s.running[channelID] = true
	s.mu.Unlock()

	s.reg.Update(channelID, func(e *registry.Entry) { e.Syncing = true })
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer func() {
			s.reg.Update(channelID, func(e *registry.Entry) { e.Syncing = false })
			s.mu.Lock()
			delete(s.running, channelID)
			s.mu.Unlock()
		}()
		select {
		case s.sem <- struct{}{}:
		case <-ctx.Done():
			return
		}
		defer func() { <-s.sem }()
		s.sync(ctx, guildID, channelID)
	}()
}

// Wait は実行中の同期の終了を待つ。
func (s *Syncer) Wait() { s.wg.Wait() }

func (s *Syncer) sync(ctx context.Context, guildID, channelID snowflake.ID) {
	cursor, ok, err := s.store.SyncedUntil(ctx, channelID)
	if err != nil {
		s.log.Error("failed to load synced position", slog.Uint64("channel_id", uint64(channelID)), slog.Any("err", err))
		return
	}
	if !ok {
		return // 同期開始前に監視解除された
	}
	attrs := []any{slog.Uint64("guild_id", uint64(guildID)), slog.Uint64("channel_id", uint64(channelID))}
	start := time.Now()
	var pages, total int
	var tracked int64
	retries := 0

	for {
		if !s.reg.IsMonitored(channelID) {
			return
		}
		msgs, err := s.api.GetMessagesAfter(ctx, channelID, cursor, pageSize)
		if err != nil {
			switch discordapi.Classify(err) {
			case discordapi.KindCanceled:
				return
			case discordapi.KindChannelGone:
				s.log.Warn("channel not found while syncing, remove settings", append(attrs, slog.Any("err", err))...)
				if err := s.store.PurgeChannel(ctx, channelID); err != nil {
					s.log.Error("failed to purge channel", slog.Any("err", err))
				}
				s.reg.Delete(channelID)
				return
			case discordapi.KindChannelDenied:
				s.log.Warn("cannot read message history, skip syncing", append(attrs, slog.Any("err", err))...)
				return
			default:
				if retries >= 3 {
					s.log.Error("failed to fetch messages, give up syncing for now", append(attrs, slog.Any("err", err))...)
					return
				}
				retries++
				select {
				case <-time.After(time.Duration(retries) * 5 * time.Second):
				case <-ctx.Done():
					return
				}
				continue
			}
		}
		retries = 0
		pages++
		total += len(msgs)

		next := cursor
		batch := make([]store.Message, 0, len(msgs))
		for _, m := range msgs {
			if m.ID > next {
				next = m.ID
			}
			if !m.Type.Deleteable() || m.Flags.Has(discord.MessageFlagEphemeral) {
				continue
			}
			batch = append(batch, store.Message{ID: m.ID, GuildID: guildID, ChannelID: channelID})
		}
		n, err := s.store.TrackMessages(ctx, batch)
		if err != nil {
			s.log.Error("failed to track synced messages", append(attrs, slog.Any("err", err))...)
			return
		}
		tracked += n
		if next > cursor {
			cursor = next
			if err := s.store.UpdateSyncedUntil(ctx, channelID, cursor); err != nil {
				s.log.Error("failed to update synced position", append(attrs, slog.Any("err", err))...)
				return
			}
		}
		if len(msgs) < pageSize {
			break
		}
		if s.maxPages > 0 && pages >= s.maxPages {
			s.log.Warn("sync page limit reached, continue on next sync", append(attrs, slog.Int("pages", pages))...)
			break
		}
	}
	if total > 0 {
		s.log.Info("channel synced", append(attrs,
			slog.Int("pages", pages), slog.Int("fetched", total), slog.Int64("tracked", tracked),
			slog.Duration("elapsed", time.Since(start)))...)
	}
}
