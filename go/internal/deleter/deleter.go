// Package deleter は削除予定日時を過ぎたメッセージを削除する。
//
// 処理の流れ:
//
//	[毎秒] DB から期限切れのメッセージをリース付きで取得（ClaimDue）
//	   ↓ チャンネルごとのメモリ内キューに振り分け
//	[チャンネルワーカー] 1チャンネルにつき1ゴルーチンで直列に削除
//	   ・投稿から14日以内のもの … 一括削除 API で最大100件/リクエスト
//	   ・14日を超えたもの       … 1件ずつ削除 API
//	   ↓
//	[結果] 成功: 記録削除 / 一時エラー: 指数バックオフで再試行 / 権限エラー: チャンネルごと保留
//
// レートリミットは「チャンネル（= Discord のバケット単位）ごとに直列化」「同時ワーカー数の上限」
// 「API 層のグローバルトークンバケット」「disgo のバケット追従」で多重に抑える。
// 同時刻に大量の削除予定があっても、DB から取り出す量をキュー上限で制限するため
// メモリやリクエストが溢れることはなく、処理しきれない分は DB に残って順次処理される。
package deleter

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/disgoorg/snowflake/v2"

	"github.com/varubogu/discord_message_cleaner_go/internal/discordapi"
	"github.com/varubogu/discord_message_cleaner_go/internal/registry"
	"github.com/varubogu/discord_message_cleaner_go/internal/store"
)

// BulkDeleteMaxAge は一括削除 API で削除できるメッセージの最大経過時間（Discord の仕様で14日）。
const BulkDeleteMaxAge = 14 * 24 * time.Hour

// maxSinglesPerBatch は1バッチに含める個別削除（14日超）の最大数。
const maxSinglesPerBatch = 10

// bulkSafetyMargin は時計のずれやキュー待ちで14日を超えてしまうのを避けるための余裕。
const bulkSafetyMargin = 10 * time.Minute

// Store は Deleter が使う DB 操作。
type Store interface {
	ClaimDue(ctx context.Context, limit int, lease time.Duration, excludeChannels []snowflake.ID) ([]store.Claimed, error)
	RetryLater(ctx context.Context, ids []snowflake.ID, retryAt time.Time, reason string, maxAttempts int) (int64, error)
	BackoffChannel(ctx context.Context, channelID snowflake.ID, until time.Time, reason string) error
	RecordChannelFailure(ctx context.Context, channelID snowflake.ID, reason string) (int, bool, error)
	ResetChannelFailure(ctx context.Context, channelID snowflake.ID) error
	PurgeChannel(ctx context.Context, channelID snowflake.ID) error
	DeleteChannel(ctx context.Context, guildID, channelID snowflake.ID) (bool, error)
	DropChannelTracked(ctx context.Context, channelID snowflake.ID) error
	ReleaseClaims(ctx context.Context, ids []snowflake.ID) error
}

// Discord は削除に使う Discord API。
type Discord interface {
	DeleteMessage(ctx context.Context, channelID, messageID snowflake.ID) error
	BulkDeleteMessages(ctx context.Context, channelID snowflake.ID, messageIDs []snowflake.ID) error
}

// Remover は削除済みメッセージの記録を消す（tracker.Tracker）。
type Remover interface {
	Remove(ctx context.Context, id snowflake.ID)
}

// Config は Deleter の設定。
type Config struct {
	Interval              time.Duration
	ClaimBatchSize        int
	ClaimLease            time.Duration
	ChannelQueueLimit     int
	Workers               int
	MaxAttempts           int
	ChannelFailureLimit   int
	ChannelFailureBackoff time.Duration
}

type channelQueue struct {
	guildID snowflake.ID
	items   []store.Claimed
	running bool
}

// Deleter は削除スケジューラとチャンネルワーカーの集合。
type Deleter struct {
	cfg     Config
	store   Store
	api     Discord
	remover Remover
	reg     *registry.Registry
	log     *slog.Logger
	now     func() time.Time

	mu       sync.Mutex
	queues   map[snowflake.ID]*channelQueue
	inflight map[snowflake.ID]snowflake.ID // messageID -> channelID
	sem      chan struct{}
	wg       sync.WaitGroup
	wake     chan struct{}
}

// New は Deleter を作成する。
func New(cfg Config, s Store, api Discord, remover Remover, reg *registry.Registry, log *slog.Logger) *Deleter {
	return &Deleter{
		cfg:      cfg,
		store:    s,
		api:      api,
		remover:  remover,
		reg:      reg,
		log:      log.With(slog.String("component", "deleter")),
		now:      time.Now,
		queues:   map[snowflake.ID]*channelQueue{},
		inflight: map[snowflake.ID]snowflake.ID{},
		sem:      make(chan struct{}, cfg.Workers),
		wake:     make(chan struct{}, 1),
	}
}

// Wake は次の周期を待たずに DB を確認させる（/clear 実行直後など）。
func (d *Deleter) Wake() {
	select {
	case d.wake <- struct{}{}:
	default:
	}
}

// Run は ctx が終了するまで毎周期 DB を確認して削除を行う。
func (d *Deleter) Run(ctx context.Context) {
	ticker := time.NewTicker(d.cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
		case <-d.wake:
		case <-ctx.Done():
			d.shutdown()
			return
		}
		if err := d.Tick(ctx); err != nil && ctx.Err() == nil {
			d.log.Error("tick failed", slog.Any("err", err))
		}
	}
}

// Tick は1周期分の処理（DB から取得してキューに積む）を行う。
func (d *Deleter) Tick(ctx context.Context) error {
	full := d.fullChannels()
	items, err := d.store.ClaimDue(ctx, d.cfg.ClaimBatchSize, d.cfg.ClaimLease, full)
	if err != nil {
		return fmt.Errorf("claim due messages: %w", err)
	}
	if len(items) > 0 {
		d.log.Debug("claimed due messages", slog.Int("count", len(items)), slog.Int("skipped_channels", len(full)))
	}
	d.enqueue(ctx, items)
	return nil
}

func (d *Deleter) fullChannels() []snowflake.ID {
	d.mu.Lock()
	defer d.mu.Unlock()
	var out []snowflake.ID
	for ch, q := range d.queues {
		if len(q.items) >= d.cfg.ChannelQueueLimit {
			out = append(out, ch)
		}
	}
	return out
}

func (d *Deleter) enqueue(ctx context.Context, items []store.Claimed) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, it := range items {
		if _, ok := d.inflight[it.MessageID]; ok {
			continue // リース切れで再取得されたがまだキューにあるもの
		}
		q, ok := d.queues[it.ChannelID]
		if !ok {
			q = &channelQueue{guildID: it.GuildID}
			d.queues[it.ChannelID] = q
		}
		q.items = append(q.items, it)
		d.inflight[it.MessageID] = it.ChannelID
		if !q.running {
			q.running = true
			d.wg.Add(1)
			go d.worker(ctx, it.ChannelID)
		}
	}
}

// take はチャンネルのキューから最大 MaxBulkDelete 件取り出す。空ならワーカーを終了扱いにする。
func (d *Deleter) take(channelID snowflake.ID) []store.Claimed {
	d.mu.Lock()
	defer d.mu.Unlock()
	q, ok := d.queues[channelID]
	if !ok {
		return nil
	}
	if len(q.items) == 0 {
		q.running = false
		delete(d.queues, channelID)
		return nil
	}
	// 14日超のメッセージは1件ずつのリクエストになるため、1バッチに含める数を抑えて
	// 同時実行枠（sem）を長時間占有しないようにする
	cutoff := d.now().Add(-BulkDeleteMaxAge + bulkSafetyMargin)
	n, old := 0, 0
	for n < len(q.items) && n < discordapi.MaxBulkDelete {
		if !q.items[n].PostedAt.After(cutoff) {
			if old >= maxSinglesPerBatch {
				break
			}
			old++
		}
		n++
	}
	batch := append([]store.Claimed(nil), q.items[:n]...)
	q.items = q.items[n:]
	return batch
}

func (d *Deleter) worker(ctx context.Context, channelID snowflake.ID) {
	defer d.wg.Done()
	for {
		// 同時にリクエストを投げるチャンネル数を制限する。バッチごとに解放して他チャンネルにも順番を回す。
		select {
		case d.sem <- struct{}{}:
		case <-ctx.Done():
			return
		}
		batch := d.take(channelID)
		if batch == nil {
			<-d.sem
			return
		}
		d.process(ctx, channelID, batch)
		<-d.sem
		if ctx.Err() != nil {
			return
		}
	}
}

// process は1バッチ分を削除する。
func (d *Deleter) process(ctx context.Context, channelID snowflake.ID, batch []store.Claimed) {
	cutoff := d.now().Add(-BulkDeleteMaxAge + bulkSafetyMargin)
	var recent, old []store.Claimed
	for _, it := range batch {
		if it.PostedAt.After(cutoff) {
			recent = append(recent, it)
		} else {
			old = append(old, it)
		}
	}

	if len(recent) >= discordapi.MinBulkDelete {
		err := d.api.BulkDeleteMessages(ctx, channelID, ids(recent))
		switch k := discordapi.Classify(err); {
		case err == nil:
			d.succeeded(ctx, channelID, recent)
		case k == discordapi.KindBadBulk || k == discordapi.KindMessageGone || k == discordapi.KindUndeletable:
			// 一括削除の条件を満たさないものが含まれていた場合は1件ずつ削除する
			d.log.Warn("bulk delete rejected, fallback to single delete", slog.Uint64("channel_id", uint64(channelID)), slog.Any("err", err))
			old = append(recent, old...)
		default:
			if !d.failed(ctx, channelID, recent, err) {
				d.release(old)
				return
			}
		}
	} else {
		old = append(recent, old...)
	}

	for i, it := range old {
		one := old[i : i+1]
		err := d.api.DeleteMessage(ctx, channelID, it.MessageID)
		if err == nil {
			d.succeeded(ctx, channelID, one)
			continue
		}
		switch discordapi.Classify(err) {
		case discordapi.KindMessageGone:
			// 既に削除済み
			d.succeeded(ctx, channelID, one)
			continue
		case discordapi.KindUndeletable, discordapi.KindBadBulk:
			d.log.Info("message cannot be deleted, drop it", slog.Uint64("channel_id", uint64(channelID)), slog.Uint64("message_id", uint64(it.MessageID)), slog.Any("err", err))
			d.succeeded(ctx, channelID, one)
			continue
		}
		if !d.failed(ctx, channelID, one, err) {
			d.release(old[i+1:])
			return
		}
	}
}

// succeeded は削除完了（または削除不要）になったメッセージの記録を消す。
func (d *Deleter) succeeded(ctx context.Context, channelID snowflake.ID, items []store.Claimed) {
	for _, it := range items {
		d.remover.Remove(ctx, it.MessageID)
	}
	d.release(items)

	if e, ok := d.reg.Get(channelID); ok && e.FailureCount > 0 {
		if err := d.store.ResetChannelFailure(ctx, channelID); err != nil {
			d.log.Warn("failed to reset channel failure count", slog.Any("err", err))
			return
		}
		d.reg.Update(channelID, func(e *registry.Entry) { e.FailureCount = 0 })
		d.log.Info("channel recovered, failure count reset", slog.Uint64("channel_id", uint64(channelID)))
	}
}

// failed はエラーを処理する。チャンネルの処理を続けてよければ true を返す。
func (d *Deleter) failed(ctx context.Context, channelID snowflake.ID, items []store.Claimed, err error) bool {
	kind := discordapi.Classify(err)
	logAttrs := []any{slog.Uint64("channel_id", uint64(channelID)), slog.Int("count", len(items)), slog.Any("err", err)}

	switch kind {
	case discordapi.KindCanceled:
		// シャットダウン中。リースは終了時にまとめて解放する。
		return false

	case discordapi.KindChannelGone:
		d.log.Warn("channel not found, remove all settings for the channel", logAttrs...)
		if err := d.store.PurgeChannel(ctx, channelID); err != nil {
			d.log.Error("failed to purge channel", slog.Any("err", err))
		}
		d.reg.Delete(channelID)
		d.release(items)
		d.dropQueue(channelID)
		return false

	case discordapi.KindChannelDenied:
		d.channelDenied(ctx, channelID, err)
		d.release(items)
		d.dropQueue(channelID)
		return false

	default: // KindTemporary
		attempts := 0
		for _, it := range items {
			attempts = max(attempts, it.Attempts)
		}
		retryAt := d.now().Add(retryBackoff(attempts))
		dropped, rerr := d.store.RetryLater(ctx, ids(items), retryAt, err.Error(), d.cfg.MaxAttempts)
		if rerr != nil {
			d.log.Error("failed to schedule retry", slog.Any("err", rerr))
		}
		if dropped > 0 {
			d.log.Error("gave up deleting messages after max attempts", append(logAttrs, slog.Int64("dropped", dropped))...)
		} else {
			d.log.Warn("temporary error, retry later", append(logAttrs, slog.Time("retry_at", retryAt))...)
		}
		d.release(items)
		return true
	}
}

// channelDenied は権限不足などチャンネル単位のエラーを処理する。
// 同じチャンネルへ失敗リクエストを繰り返すと Discord の不正リクエスト制限（10分で1万回）に近づくため、
// チャンネル全体を一定時間保留し、規定回数連続したら監視を解除する。
func (d *Deleter) channelDenied(ctx context.Context, channelID snowflake.ID, cause error) {
	reason := cause.Error()
	until := d.now().Add(d.cfg.ChannelFailureBackoff)
	if err := d.store.BackoffChannel(ctx, channelID, until, reason); err != nil {
		d.log.Error("failed to backoff channel", slog.Any("err", err))
	}
	count, monitored, err := d.store.RecordChannelFailure(ctx, channelID, reason)
	if err != nil {
		d.log.Error("failed to record channel failure", slog.Any("err", err))
		return
	}
	attrs := []any{slog.Uint64("channel_id", uint64(channelID)), slog.String("reason", reason)}
	if !monitored {
		// 監視対象外（/clear で予約したもの）は再試行しない
		d.log.Warn("cannot delete messages in unmonitored channel, drop scheduled deletions", attrs...)
		if err := d.store.DropChannelTracked(ctx, channelID); err != nil {
			d.log.Error("failed to drop tracked messages", slog.Any("err", err))
		}
		return
	}
	d.reg.Update(channelID, func(e *registry.Entry) { e.FailureCount = count })
	if count >= d.cfg.ChannelFailureLimit {
		e, _ := d.reg.Get(channelID)
		d.log.Warn("channel failure limit exceeded, disable monitoring", append(attrs, slog.Int("failures", count))...)
		if _, err := d.store.DeleteChannel(ctx, e.GuildID, channelID); err != nil {
			d.log.Error("failed to disable channel", slog.Any("err", err))
			return
		}
		d.reg.Delete(channelID)
		return
	}
	d.log.Warn("cannot delete messages in channel, pause the channel", append(attrs, slog.Int("failures", count), slog.Time("until", until))...)
}

func (d *Deleter) release(items []store.Claimed) {
	if len(items) == 0 {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, it := range items {
		delete(d.inflight, it.MessageID)
	}
}

// dropQueue はチャンネルのキューを破棄する（DB 側はバックオフ済み・削除済み）。
func (d *Deleter) dropQueue(channelID snowflake.ID) {
	d.mu.Lock()
	defer d.mu.Unlock()
	q, ok := d.queues[channelID]
	if !ok {
		return
	}
	for _, it := range q.items {
		delete(d.inflight, it.MessageID)
	}
	q.items = nil
}

// DropChannel は監視解除時などに、そのチャンネルの処理待ちを破棄する。
func (d *Deleter) DropChannel(channelID snowflake.ID) { d.dropQueue(channelID) }

// shutdown はワーカーの終了を待ち、未処理のリースを解放する。
func (d *Deleter) shutdown() {
	d.wg.Wait()
	d.mu.Lock()
	pending := make([]snowflake.ID, 0, len(d.inflight))
	for id := range d.inflight {
		pending = append(pending, id)
	}
	d.mu.Unlock()
	if len(pending) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := d.store.ReleaseClaims(ctx, pending); err != nil {
		d.log.Warn("failed to release claims", slog.Any("err", err))
		return
	}
	d.log.Info("released pending claims", slog.Int("count", len(pending)))
}

// Stats は処理待ち件数を返す（ログ・確認用）。
func (d *Deleter) Stats() (channels, pending int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.queues), len(d.inflight)
}

func retryBackoff(attempts int) time.Duration {
	b := 10 * time.Second
	for range attempts {
		b *= 2
		if b >= 10*time.Minute {
			return 10 * time.Minute
		}
	}
	return b
}

func ids(items []store.Claimed) []snowflake.ID {
	out := make([]snowflake.ID, len(items))
	for i, it := range items {
		out[i] = it.MessageID
	}
	return out
}
