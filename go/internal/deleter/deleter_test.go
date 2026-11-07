package deleter

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"

	"github.com/varubogu/discord_message_cleaner_go/internal/registry"
	"github.com/varubogu/discord_message_cleaner_go/internal/store"
)

type fakeStore struct {
	mu        sync.Mutex
	due       []store.Claimed
	retried   []snowflake.ID
	backoffs  []snowflake.ID
	failures  map[snowflake.ID]int
	disabled  []snowflake.ID
	purged    []snowflake.ID
	monitored bool
}

func (s *fakeStore) ClaimDue(_ context.Context, limit int, _ time.Duration, exclude []snowflake.ID) ([]store.Claimed, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out, rest []store.Claimed
	for _, c := range s.due {
		if len(out) < limit && !slices.Contains(exclude, c.ChannelID) {
			out = append(out, c)
		} else {
			rest = append(rest, c)
		}
	}
	s.due = rest
	return out, nil
}

func (s *fakeStore) RetryLater(_ context.Context, ids []snowflake.ID, _ time.Time, _ string, _ int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.retried = append(s.retried, ids...)
	return 0, nil
}

func (s *fakeStore) BackoffChannel(_ context.Context, ch snowflake.ID, _ time.Time, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.backoffs = append(s.backoffs, ch)
	return nil
}

func (s *fakeStore) RecordChannelFailure(_ context.Context, ch snowflake.ID, _ string) (int, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failures == nil {
		s.failures = map[snowflake.ID]int{}
	}
	s.failures[ch]++
	return s.failures[ch], s.monitored, nil
}

func (s *fakeStore) ResetChannelFailure(context.Context, snowflake.ID) error { return nil }

func (s *fakeStore) PurgeChannel(_ context.Context, ch snowflake.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purged = append(s.purged, ch)
	return nil
}

func (s *fakeStore) DeleteChannel(_ context.Context, _, ch snowflake.ID) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.disabled = append(s.disabled, ch)
	return true, nil
}

func (s *fakeStore) DropChannelTracked(context.Context, snowflake.ID) error { return nil }
func (s *fakeStore) ReleaseClaims(context.Context, []snowflake.ID) error    { return nil }

type fakeAPI struct {
	mu         sync.Mutex
	bulkCalls  [][]snowflake.ID
	singles    []snowflake.ID
	bulkErr    error
	singleErr  func(id snowflake.ID) error
	active     atomic.Int32
	maxActive  atomic.Int32
	callDelay  time.Duration
	totalCalls atomic.Int32
}

func (a *fakeAPI) enter() func() {
	n := a.active.Add(1)
	for {
		m := a.maxActive.Load()
		if n <= m || a.maxActive.CompareAndSwap(m, n) {
			break
		}
	}
	a.totalCalls.Add(1)
	if a.callDelay > 0 {
		time.Sleep(a.callDelay)
	}
	return func() { a.active.Add(-1) }
}

func (a *fakeAPI) DeleteMessage(_ context.Context, _, id snowflake.ID) error {
	defer a.enter()()
	a.mu.Lock()
	a.singles = append(a.singles, id)
	a.mu.Unlock()
	if a.singleErr != nil {
		return a.singleErr(id)
	}
	return nil
}

func (a *fakeAPI) BulkDeleteMessages(_ context.Context, _ snowflake.ID, ids []snowflake.ID) error {
	defer a.enter()()
	a.mu.Lock()
	a.bulkCalls = append(a.bulkCalls, slices.Clone(ids))
	a.mu.Unlock()
	return a.bulkErr
}

type fakeRemover struct {
	mu      sync.Mutex
	removed []snowflake.ID
}

func (r *fakeRemover) Remove(_ context.Context, id snowflake.ID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.removed = append(r.removed, id)
}

func newTestDeleter(s *fakeStore, api *fakeAPI, rm *fakeRemover, reg *registry.Registry) *Deleter {
	return New(Config{
		Interval:              time.Second,
		ClaimBatchSize:        1000,
		ClaimLease:            time.Minute,
		ChannelQueueLimit:     500,
		Workers:               4,
		MaxAttempts:           5,
		ChannelFailureLimit:   3,
		ChannelFailureBackoff: time.Minute,
	}, s, api, rm, reg, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func waitIdle(t *testing.T, d *Deleter) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if ch, p := d.Stats(); ch == 0 && p == 0 {
			d.wg.Wait()
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("deleter did not become idle")
}

// 削除対象を作る。ageは投稿からの経過時間。
func claimed(id, ch uint64, age time.Duration) store.Claimed {
	now := time.Now()
	return store.Claimed{
		MessageID: snowflake.ID(id),
		GuildID:   1,
		ChannelID: snowflake.ID(ch),
		PostedAt:  now.Add(-age),
		DeleteAt:  now.Add(-time.Second),
	}
}

func TestBulkAndSingleSplit(t *testing.T) {
	s := &fakeStore{}
	for i := range 150 {
		s.due = append(s.due, claimed(uint64(1000+i), 10, time.Hour))
	}
	for i := range 3 {
		s.due = append(s.due, claimed(uint64(5000+i), 10, 20*24*time.Hour)) // 14日超
	}
	api := &fakeAPI{}
	rm := &fakeRemover{}
	d := newTestDeleter(s, api, rm, registry.New())

	if err := d.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, d)

	if len(api.bulkCalls) != 2 || len(api.bulkCalls[0]) != 100 || len(api.bulkCalls[1]) != 50 {
		t.Fatalf("unexpected bulk calls: %d %v", len(api.bulkCalls), lens(api.bulkCalls))
	}
	if len(api.singles) != 3 {
		t.Fatalf("expected 3 single deletes, got %d", len(api.singles))
	}
	if len(rm.removed) != 153 {
		t.Fatalf("expected 153 removed, got %d", len(rm.removed))
	}
	// 153件の削除が 5 リクエストで済む
	if got := api.totalCalls.Load(); got != 5 {
		t.Fatalf("expected 5 requests, got %d", got)
	}
}

func TestSingleRecentMessageUsesSingleDelete(t *testing.T) {
	s := &fakeStore{due: []store.Claimed{claimed(1, 10, time.Minute)}}
	api := &fakeAPI{}
	d := newTestDeleter(s, api, &fakeRemover{}, registry.New())
	_ = d.Tick(context.Background())
	waitIdle(t, d)
	if len(api.bulkCalls) != 0 || len(api.singles) != 1 {
		t.Fatalf("bulk=%d singles=%d", len(api.bulkCalls), len(api.singles))
	}
}

func TestWorkerConcurrencyLimit(t *testing.T) {
	s := &fakeStore{}
	for ch := range 20 {
		for i := range 3 {
			s.due = append(s.due, claimed(uint64(ch*100+i+1), uint64(ch+1), time.Hour))
		}
	}
	api := &fakeAPI{callDelay: 10 * time.Millisecond}
	d := newTestDeleter(s, api, &fakeRemover{}, registry.New())
	_ = d.Tick(context.Background())
	waitIdle(t, d)
	if got := api.maxActive.Load(); got > 4 {
		t.Fatalf("max concurrent requests %d exceeds workers 4", got)
	}
	if len(api.bulkCalls) != 20 {
		t.Fatalf("expected 20 bulk calls (1 per channel), got %d", len(api.bulkCalls))
	}
}

func TestChannelDeniedBacksOffAndDisables(t *testing.T) {
	reg := registry.New()
	reg.Set(registry.Entry{GuildID: 1, ChannelID: 10, Lifetime: time.Hour})
	s := &fakeStore{monitored: true}
	api := &fakeAPI{bulkErr: restErr(http.StatusForbidden, rest.JSONErrorCodeLackPermissionsToPerformAction)}
	d := newTestDeleter(s, api, &fakeRemover{}, reg)

	for round := 1; round <= 3; round++ {
		for i := range 5 {
			s.due = append(s.due, claimed(uint64(round*100+i), 10, time.Hour))
		}
		_ = d.Tick(context.Background())
		waitIdle(t, d)
		if len(api.bulkCalls) != round {
			t.Fatalf("round %d: after a permission error the rest of the channel must not be requested (bulk calls=%d)", round, len(api.bulkCalls))
		}
	}
	if len(s.backoffs) != 3 {
		t.Fatalf("expected 3 backoffs, got %d", len(s.backoffs))
	}
	if len(s.disabled) != 1 || reg.IsMonitored(10) {
		t.Fatalf("channel should be disabled after 3 failures: disabled=%v", s.disabled)
	}
}

func TestTemporaryErrorRetries(t *testing.T) {
	s := &fakeStore{}
	for i := range 3 {
		s.due = append(s.due, claimed(uint64(i+1), 10, time.Hour))
	}
	api := &fakeAPI{bulkErr: restErr(http.StatusInternalServerError, 0)}
	rm := &fakeRemover{}
	d := newTestDeleter(s, api, rm, registry.New())
	_ = d.Tick(context.Background())
	waitIdle(t, d)
	if len(s.retried) != 3 || len(rm.removed) != 0 {
		t.Fatalf("retried=%d removed=%d", len(s.retried), len(rm.removed))
	}
}

func TestUnknownMessageTreatedAsDeleted(t *testing.T) {
	s := &fakeStore{due: []store.Claimed{claimed(1, 10, 30*24*time.Hour)}}
	api := &fakeAPI{singleErr: func(snowflake.ID) error {
		return restErr(http.StatusNotFound, rest.JSONErrorCodeUnknownMessage)
	}}
	rm := &fakeRemover{}
	d := newTestDeleter(s, api, rm, registry.New())
	_ = d.Tick(context.Background())
	waitIdle(t, d)
	if len(rm.removed) != 1 || len(s.retried) != 0 {
		t.Fatalf("removed=%d retried=%d", len(rm.removed), len(s.retried))
	}
}

func TestUnknownChannelPurges(t *testing.T) {
	reg := registry.New()
	reg.Set(registry.Entry{GuildID: 1, ChannelID: 10, Lifetime: time.Hour})
	s := &fakeStore{due: []store.Claimed{claimed(1, 10, time.Hour), claimed(2, 10, time.Hour)}}
	api := &fakeAPI{bulkErr: restErr(http.StatusNotFound, rest.JSONErrorCodeUnknownChannel)}
	d := newTestDeleter(s, api, &fakeRemover{}, reg)
	_ = d.Tick(context.Background())
	waitIdle(t, d)
	if len(s.purged) != 1 || reg.IsMonitored(10) {
		t.Fatalf("purged=%v", s.purged)
	}
}

func TestBulkRejectedFallsBackToSingle(t *testing.T) {
	s := &fakeStore{due: []store.Claimed{claimed(1, 10, time.Hour), claimed(2, 10, time.Hour)}}
	api := &fakeAPI{bulkErr: restErr(http.StatusBadRequest, rest.JSONErrorCodeMessageTooOldToBulkDelete)}
	rm := &fakeRemover{}
	d := newTestDeleter(s, api, rm, registry.New())
	_ = d.Tick(context.Background())
	waitIdle(t, d)
	if len(api.singles) != 2 || len(rm.removed) != 2 {
		t.Fatalf("singles=%d removed=%d", len(api.singles), len(rm.removed))
	}
}

func TestQueueLimitExcludesChannelFromClaim(t *testing.T) {
	s := &fakeStore{}
	api := &fakeAPI{}
	d := newTestDeleter(s, api, &fakeRemover{}, registry.New())
	d.cfg.ChannelQueueLimit = 2
	// ワーカーを起動させないよう、直接キューを積む
	d.queues[10] = &channelQueue{guildID: 1, items: []store.Claimed{claimed(1, 10, 0), claimed(2, 10, 0)}, running: true}
	if full := d.fullChannels(); len(full) != 1 || full[0] != 10 {
		t.Fatalf("full=%v", full)
	}
}

func TestDuplicateClaimIgnored(t *testing.T) {
	api := &fakeAPI{}
	d := newTestDeleter(&fakeStore{}, api, &fakeRemover{}, registry.New())
	c := claimed(1, 10, time.Hour)
	d.mu.Lock()
	d.inflight[c.MessageID] = c.ChannelID
	d.mu.Unlock()
	d.enqueue(context.Background(), []store.Claimed{c})
	if _, ok := d.queues[10]; ok {
		t.Fatal("duplicate must not be enqueued")
	}
}

func TestRetryBackoff(t *testing.T) {
	if retryBackoff(0) != 10*time.Second || retryBackoff(1) != 20*time.Second || retryBackoff(20) != 10*time.Minute {
		t.Fatal("unexpected backoff")
	}
}

func restErr(status int, code rest.JSONErrorCode) error {
	return &rest.Error{Code: code, Message: "test", Response: &http.Response{StatusCode: status, Status: http.StatusText(status)}}
}

func lens(x [][]snowflake.ID) []int {
	out := make([]int, len(x))
	for i, v := range x {
		out[i] = len(v)
	}
	return out
}
