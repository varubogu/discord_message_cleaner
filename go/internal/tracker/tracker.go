// Package tracker はメッセージ投稿・削除イベントをまとめて DB に書き込む。
//
// イベント1件ごとに DB へ書き込むと負荷が高いため、一定時間または一定件数ごとに
// 1つの SQL（unnest による一括 INSERT / DELETE）にまとめて反映する。
package tracker

import (
	"context"
	"log/slog"
	"time"

	"github.com/disgoorg/snowflake/v2"

	"github.com/varubogu/discord_message_cleaner_go/internal/registry"
	"github.com/varubogu/discord_message_cleaner_go/internal/store"
)

// Store は Tracker が使う DB 操作。
type Store interface {
	TrackMessages(ctx context.Context, msgs []store.Message) (int64, error)
	RemoveTracked(ctx context.Context, ids []snowflake.ID) error
	UpdateSyncedUntilBatch(ctx context.Context, latest map[snowflake.ID]snowflake.ID) error
}

type opKind int

const (
	opTrack opKind = iota
	opRemove
)

type op struct {
	kind opKind
	msg  store.Message
}

// Tracker はイベントをバッファして DB に一括反映する。
type Tracker struct {
	store         Store
	reg           *registry.Registry
	ops           chan op
	flushInterval time.Duration
	batchSize     int
	maxPending    int
	log           *slog.Logger

	tracks  []store.Message
	removes []snowflake.ID
}

// New は Tracker を作成する。
func New(s Store, reg *registry.Registry, flushInterval time.Duration, batchSize, queueSize int, log *slog.Logger) *Tracker {
	return &Tracker{
		store:         s,
		reg:           reg,
		ops:           make(chan op, queueSize),
		flushInterval: flushInterval,
		batchSize:     batchSize,
		maxPending:    max(batchSize*20, queueSize),
		log:           log.With(slog.String("component", "tracker")),
	}
}

// Track は投稿されたメッセージを記録対象としてキューに積む。
func (t *Tracker) Track(ctx context.Context, m store.Message) {
	t.enqueue(ctx, op{kind: opTrack, msg: m})
}

// Remove は削除されたメッセージを記録から外すようキューに積む。
func (t *Tracker) Remove(ctx context.Context, id snowflake.ID) {
	t.enqueue(ctx, op{kind: opRemove, msg: store.Message{ID: id}})
}

func (t *Tracker) enqueue(ctx context.Context, o op) {
	select {
	case t.ops <- o:
	case <-ctx.Done():
	}
}

// Run はキューを処理する。ctx が終了すると残りを書き込んでから戻る。
func (t *Tracker) Run(ctx context.Context) {
	ticker := time.NewTicker(t.flushInterval)
	defer ticker.Stop()
	for {
		select {
		case o := <-t.ops:
			t.add(o)
			if len(t.tracks)+len(t.removes) >= t.batchSize {
				t.flush(ctx)
			}
		case <-ticker.C:
			t.flush(ctx)
		case <-ctx.Done():
			t.drain()
			return
		}
	}
}

func (t *Tracker) add(o op) {
	switch o.kind {
	case opTrack:
		t.tracks = append(t.tracks, o.msg)
	case opRemove:
		t.removes = append(t.removes, o.msg.ID)
	}
}

func (t *Tracker) drain() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for {
		select {
		case o := <-t.ops:
			t.add(o)
		default:
			t.flush(ctx)
			return
		}
	}
}

func (t *Tracker) flush(ctx context.Context) {
	if len(t.tracks) > 0 {
		n, err := t.store.TrackMessages(ctx, t.tracks)
		if err != nil {
			t.log.Error("failed to track messages", slog.Int("count", len(t.tracks)), slog.Any("err", err))
			var dropped int
			if t.tracks, dropped = trimPending(t.tracks, t.maxPending); dropped > 0 {
				t.log.Error("dropped pending tracks", slog.Int("count", dropped))
			}
		} else {
			t.log.Debug("tracked messages", slog.Int("received", len(t.tracks)), slog.Int64("inserted", n))
			// 同期済み位置を進める。DBへの書き込みに失敗した分は進めないので、次回起動時の同期で回収される。
			latest := map[snowflake.ID]snowflake.ID{}
			for _, m := range t.tracks {
				if e, ok := t.reg.Get(m.ChannelID); !ok || e.Syncing {
					continue
				}
				if m.ID > latest[m.ChannelID] {
					latest[m.ChannelID] = m.ID
				}
			}
			if err := t.store.UpdateSyncedUntilBatch(ctx, latest); err != nil {
				t.log.Warn("failed to update synced position", slog.Any("err", err))
			}
			t.tracks = t.tracks[:0]
		}
	}
	if len(t.removes) > 0 {
		if err := t.store.RemoveTracked(ctx, t.removes); err != nil {
			t.log.Error("failed to remove tracked messages", slog.Int("count", len(t.removes)), slog.Any("err", err))
			var dropped int
			if t.removes, dropped = trimPending(t.removes, t.maxPending); dropped > 0 {
				t.log.Error("dropped pending removes", slog.Int("count", dropped))
			}
		} else {
			t.removes = t.removes[:0]
		}
	}
}

// trimPending は DB 障害時にメモリを使い切らないよう、古いものから捨てる。
func trimPending[T any](s []T, limit int) ([]T, int) {
	if len(s) <= limit {
		return s, 0
	}
	drop := len(s) - limit
	return append(s[:0:0], s[drop:]...), drop
}
