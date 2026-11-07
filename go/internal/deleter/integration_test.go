package deleter_test

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"golang.org/x/time/rate"

	"github.com/varubogu/discord_message_cleaner_go/internal/deleter"
	"github.com/varubogu/discord_message_cleaner_go/internal/registry"
	"github.com/varubogu/discord_message_cleaner_go/internal/store"
	"github.com/varubogu/discord_message_cleaner_go/internal/tracker"
)

// countingAPI は Discord API の代わり。グローバル上限を模したリミッタを通し、リクエスト数を数える。
type countingAPI struct {
	limiter  *rate.Limiter
	requests atomic.Int64
	deleted  sync.Map
	perChan  sync.Map // channelID -> *sync.Mutex（同一チャンネルへの同時リクエストを検出する）
	overlap  atomic.Int64
}

func (a *countingAPI) lockChannel(ch snowflake.ID) func() {
	v, _ := a.perChan.LoadOrStore(ch, &sync.Mutex{})
	mu := v.(*sync.Mutex)
	if !mu.TryLock() {
		a.overlap.Add(1)
		mu.Lock()
	}
	return mu.Unlock
}

func (a *countingAPI) DeleteMessage(ctx context.Context, ch, id snowflake.ID) error {
	defer a.lockChannel(ch)()
	if err := a.limiter.Wait(ctx); err != nil {
		return err
	}
	a.requests.Add(1)
	a.deleted.Store(id, true)
	return nil
}

func (a *countingAPI) BulkDeleteMessages(ctx context.Context, ch snowflake.ID, ids []snowflake.ID) error {
	defer a.lockChannel(ch)()
	if err := a.limiter.Wait(ctx); err != nil {
		return err
	}
	a.requests.Add(1)
	for _, id := range ids {
		a.deleted.Store(id, true)
	}
	return nil
}

// 同一時刻に大量の削除予定が発生しても、少ないリクエスト数・チャンネル単位の直列化で処理しきれることを確認する。
func TestManyDeletionsAtSameTime(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	st, err := store.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ListChannels(ctx); err != nil {
		t.Fatal(err)
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	reg := registry.New()
	const guild snowflake.ID = 777
	const channels = 5
	const perChannel = 2000 // 合計 10,000 件が同じ瞬間に期限切れになる
	const oldPerChannel = 5 // 14日超のもの（個別削除になる）
	_ = st.PurgeGuild(ctx, guild)

	postedAt := time.Now().Add(-2 * time.Hour)
	var msgs []store.Message
	for c := range channels {
		ch := snowflake.ID(9000 + c)
		if _, err := st.UpsertChannel(ctx, guild, ch, time.Hour, 0); err != nil {
			t.Fatal(err)
		}
		reg.Set(registry.Entry{GuildID: guild, ChannelID: ch, Lifetime: time.Hour})
		for i := range perChannel {
			msgs = append(msgs, store.Message{ID: snowflake.New(postedAt) + snowflake.ID(c*perChannel+i), GuildID: guild, ChannelID: ch})
		}
		for i := range oldPerChannel {
			msgs = append(msgs, store.Message{ID: snowflake.New(postedAt.Add(-30*24*time.Hour)) + snowflake.ID(c*100+i), GuildID: guild, ChannelID: ch})
		}
	}
	for i := 0; i < len(msgs); i += 1000 {
		if _, err := st.TrackMessages(ctx, msgs[i:min(i+1000, len(msgs))]); err != nil {
			t.Fatal(err)
		}
	}

	tr := tracker.New(st, reg, 50*time.Millisecond, 500, 10000, log)
	trCtx, trCancel := context.WithCancel(context.Background())
	trDone := make(chan struct{})
	go func() { tr.Run(trCtx); close(trDone) }()

	api := &countingAPI{limiter: rate.NewLimiter(40, 10)}
	d := deleter.New(deleter.Config{
		Interval:              100 * time.Millisecond,
		ClaimBatchSize:        1000,
		ClaimLease:            time.Minute,
		ChannelQueueLimit:     500,
		Workers:               8,
		MaxAttempts:           5,
		ChannelFailureLimit:   3,
		ChannelFailureBackoff: time.Minute,
	}, st, api, tr, reg, log)
	dCtx, dCancel := context.WithCancel(ctx)
	dDone := make(chan struct{})
	go func() { d.Run(dCtx); close(dDone) }()

	start := time.Now()
	deadline := time.Now().Add(30 * time.Second)
	for {
		stats, err := st.ChannelStats(ctx, guild)
		if err != nil {
			t.Fatal(err)
		}
		var pending int64
		for _, s := range stats {
			pending += s.Pending
		}
		if pending == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout: %d messages still pending", pending)
		}
		time.Sleep(50 * time.Millisecond)
	}
	elapsed := time.Since(start)
	dCancel()
	<-dDone
	trCancel()
	<-trDone

	total := channels * (perChannel + oldPerChannel)
	var deleted int
	api.deleted.Range(func(_, _ any) bool { deleted++; return true })
	if deleted != total {
		t.Fatalf("deleted %d, want %d", deleted, total)
	}
	// 一括削除: 2000/100 = 20 リクエスト/チャンネル + 14日超の個別削除 5 = 25 × 5チャンネル
	want := int64(channels * (perChannel/100 + oldPerChannel))
	if got := api.requests.Load(); got > want+channels {
		t.Fatalf("too many requests: %d (expected about %d)", got, want)
	}
	if api.overlap.Load() != 0 {
		t.Fatalf("concurrent requests to the same channel detected: %d", api.overlap.Load())
	}
	t.Logf("deleted %d messages with %d requests in %v", deleted, api.requests.Load(), elapsed)
}
