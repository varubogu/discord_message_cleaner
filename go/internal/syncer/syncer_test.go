package syncer

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"

	"github.com/varubogu/discord_message_cleaner_go/internal/registry"
	"github.com/varubogu/discord_message_cleaner_go/internal/store"
)

type fakeStore struct {
	mu      sync.Mutex
	synced  snowflake.ID
	tracked []snowflake.ID
}

func (s *fakeStore) SyncedUntil(context.Context, snowflake.ID) (snowflake.ID, bool, error) {
	return s.synced, true, nil
}
func (s *fakeStore) TrackMessages(_ context.Context, msgs []store.Message) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range msgs {
		s.tracked = append(s.tracked, m.ID)
	}
	return int64(len(msgs)), nil
}
func (s *fakeStore) UpdateSyncedUntil(_ context.Context, _, id snowflake.ID) error {
	s.synced = max(s.synced, id)
	return nil
}
func (s *fakeStore) PurgeChannel(context.Context, snowflake.ID) error { return nil }

// fakeAPI は Discord と同じく「after より後の古い順に limit 件を、新しい順に並べて」返す。
type fakeAPI struct {
	msgs  []discord.Message // ID 昇順
	calls int
}

func (a *fakeAPI) GetMessagesAfter(_ context.Context, _, after snowflake.ID, limit int) ([]discord.Message, error) {
	a.calls++
	var out []discord.Message
	for _, m := range a.msgs {
		if m.ID > after && len(out) < limit {
			out = append(out, m)
		}
	}
	slices.Reverse(out)
	return out, nil
}

func TestSyncPaginates(t *testing.T) {
	api := &fakeAPI{}
	for i := 1; i <= 250; i++ {
		typ := discord.MessageTypeDefault
		if i == 7 {
			typ = discord.MessageTypeThreadStarterMessage // 削除不可
		}
		api.msgs = append(api.msgs, discord.Message{ID: snowflake.ID(1000 + i), Type: typ})
	}
	st := &fakeStore{}
	reg := registry.New()
	reg.Set(registry.Entry{GuildID: 1, ChannelID: 10, Lifetime: time.Hour})
	s := New(st, api, reg, 0, 1, slog.New(slog.NewTextHandler(io.Discard, nil)))

	s.Start(context.Background(), 1, 10)
	s.Wait()

	if len(st.tracked) != 249 {
		t.Fatalf("tracked %d, want 249", len(st.tracked))
	}
	if st.synced != 1250 {
		t.Fatalf("synced_until %d, want 1250", st.synced)
	}
	if api.calls != 3 {
		t.Fatalf("calls %d, want 3", api.calls)
	}
	if e, _ := reg.Get(10); e.Syncing {
		t.Fatal("syncing flag must be cleared")
	}

	// 2回目は新着がないので1リクエストで終わる
	api.calls = 0
	s.Start(context.Background(), 1, 10)
	s.Wait()
	if api.calls != 1 || len(st.tracked) != 249 {
		t.Fatalf("second sync: calls=%d tracked=%d", api.calls, len(st.tracked))
	}
}

func TestSyncMaxPages(t *testing.T) {
	api := &fakeAPI{}
	for i := 1; i <= 500; i++ {
		api.msgs = append(api.msgs, discord.Message{ID: snowflake.ID(i)})
	}
	st := &fakeStore{}
	reg := registry.New()
	reg.Set(registry.Entry{GuildID: 1, ChannelID: 10})
	s := New(st, api, reg, 2, 1, slog.New(slog.NewTextHandler(io.Discard, nil)))
	s.Start(context.Background(), 1, 10)
	s.Wait()
	if api.calls != 2 || st.synced != 200 {
		t.Fatalf("calls=%d synced=%d", api.calls, st.synced)
	}
}
