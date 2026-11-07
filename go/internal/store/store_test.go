package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/disgoorg/snowflake/v2"
)

// TEST_DATABASE_URL が設定されている場合のみ実行する（例: postgres://postgres@127.0.0.1:5432/dmc_test?sslmode=disable）
func openTest(t *testing.T) *Store {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	s, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	// 2回目のマイグレーションは何もしない
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(ctx, `TRUNCATE monitored_channels, excluded_messages, tracked_messages`); err != nil {
		t.Fatal(err)
	}
	return s
}

// snowflakeAt は指定時刻に投稿されたメッセージIDを作る。
func snowflakeAt(t time.Time, seq uint64) snowflake.ID {
	return snowflake.New(t) + snowflake.ID(seq)
}

func TestTrackAndClaim(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	const guild, ch, other snowflake.ID = 1, 10, 20

	if res, err := s.UpsertChannel(ctx, guild, ch, time.Hour, 0); err != nil || !res.Created {
		t.Fatalf("upsert: %+v %v", res, err)
	}

	now := time.Now()
	old := snowflakeAt(now.Add(-2*time.Hour), 1)   // 期限切れ
	fresh := snowflakeAt(now.Add(-time.Minute), 2) // まだ
	excluded := snowflakeAt(now.Add(-3*time.Hour), 3)
	unmonitored := snowflakeAt(now.Add(-3*time.Hour), 4)

	if err := s.AddExclusion(ctx, Message{ID: excluded, GuildID: guild, ChannelID: ch}); err != nil {
		t.Fatal(err)
	}
	n, err := s.TrackMessages(ctx, []Message{
		{ID: old, GuildID: guild, ChannelID: ch},
		{ID: fresh, GuildID: guild, ChannelID: ch},
		{ID: excluded, GuildID: guild, ChannelID: ch},
		{ID: unmonitored, GuildID: guild, ChannelID: other},
		{ID: old, GuildID: guild, ChannelID: ch}, // 重複
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("tracked %d, want 2", n)
	}

	claimed, err := s.ClaimDue(ctx, 100, time.Minute, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(claimed) != 1 || claimed[0].MessageID != old {
		t.Fatalf("claimed %+v", claimed)
	}
	// 投稿日時は Snowflake から算出される
	if d := claimed[0].DeleteAt.Sub(claimed[0].PostedAt); d != time.Hour {
		t.Fatalf("delete_at - posted_at = %v", d)
	}
	// リース中は再取得されない
	if again, _ := s.ClaimDue(ctx, 100, time.Minute, nil); len(again) != 0 {
		t.Fatalf("claimed again: %+v", again)
	}
	// 除外チャンネル指定
	if err := s.ReleaseClaims(ctx, []snowflake.ID{old}); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.ClaimDue(ctx, 100, time.Minute, []snowflake.ID{ch}); len(got) != 0 {
		t.Fatalf("excluded channel claimed: %+v", got)
	}
	if got, _ := s.ClaimDue(ctx, 100, time.Minute, nil); len(got) != 1 {
		t.Fatalf("released claim not claimable: %+v", got)
	}

	if err := s.RemoveTracked(ctx, []snowflake.ID{old}); err != nil {
		t.Fatal(err)
	}
	if ok, _ := s.IsTracked(ctx, old); ok {
		t.Fatal("removed message still tracked")
	}
}

func TestLifetimeChangeReschedulesWithoutRefetch(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	const guild, ch snowflake.ID = 1, 10
	if _, err := s.UpsertChannel(ctx, guild, ch, 24*time.Hour, 0); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	id := snowflakeAt(now.Add(-2*time.Hour), 1)
	if _, err := s.TrackMessages(ctx, []Message{{ID: id, GuildID: guild, ChannelID: ch}}); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.ClaimDue(ctx, 10, time.Minute, nil); len(got) != 0 {
		t.Fatal("should not be due with 24h lifetime")
	}
	res, err := s.UpsertChannel(ctx, guild, ch, time.Hour, 999)
	if err != nil {
		t.Fatal(err)
	}
	if res.Created || res.Rescheduled != 1 {
		t.Fatalf("upsert result %+v", res)
	}
	got, _ := s.ClaimDue(ctx, 10, time.Minute, nil)
	if len(got) != 1 || got[0].DeleteAt.Sub(got[0].PostedAt) != time.Hour {
		t.Fatalf("not rescheduled: %+v", got)
	}
	// 既存チャンネルの同期位置は上書きされない
	if su, _, _ := s.SyncedUntil(ctx, ch); su != 0 {
		t.Fatalf("synced_until overwritten: %v", su)
	}
}

func TestRemoveExclusionRetracks(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	const guild, ch snowflake.ID = 1, 10
	if _, err := s.UpsertChannel(ctx, guild, ch, time.Hour, 0); err != nil {
		t.Fatal(err)
	}
	id := snowflakeAt(time.Now().Add(-2*time.Hour), 1)
	m := Message{ID: id, GuildID: guild, ChannelID: ch}
	if _, err := s.TrackMessages(ctx, []Message{m}); err != nil {
		t.Fatal(err)
	}
	if err := s.AddExclusion(ctx, m); err != nil {
		t.Fatal(err)
	}
	if ok, _ := s.IsTracked(ctx, id); ok {
		t.Fatal("excluded message must not be tracked")
	}
	existed, err := s.RemoveExclusion(ctx, m)
	if err != nil || !existed {
		t.Fatalf("remove exclusion: %v %v", existed, err)
	}
	if ok, _ := s.IsTracked(ctx, id); !ok {
		t.Fatal("message must be tracked again")
	}
	if existed, _ := s.RemoveExclusion(ctx, m); existed {
		t.Fatal("second remove must report not existed")
	}
}

func TestRetryAndBackoff(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	const guild, ch snowflake.ID = 1, 10
	if _, err := s.UpsertChannel(ctx, guild, ch, time.Second, 0); err != nil {
		t.Fatal(err)
	}
	id := snowflakeAt(time.Now().Add(-time.Hour), 1)
	if _, err := s.TrackMessages(ctx, []Message{{ID: id, GuildID: guild, ChannelID: ch}}); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Second)
	for i := 1; i <= 2; i++ {
		dropped, err := s.RetryLater(ctx, []snowflake.ID{id}, past, "boom", 3)
		if err != nil || dropped != 0 {
			t.Fatalf("retry %d: %d %v", i, dropped, err)
		}
	}
	if dropped, _ := s.RetryLater(ctx, []snowflake.ID{id}, past, "boom", 3); dropped != 1 {
		t.Fatal("should be dropped at max attempts")
	}

	id2 := snowflakeAt(time.Now().Add(-time.Hour), 2)
	if _, err := s.TrackMessages(ctx, []Message{{ID: id2, GuildID: guild, ChannelID: ch}}); err != nil {
		t.Fatal(err)
	}
	if err := s.BackoffChannel(ctx, ch, time.Now().Add(time.Hour), "forbidden"); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.ClaimDue(ctx, 10, time.Minute, nil); len(got) != 0 {
		t.Fatal("backed off channel must not be claimed")
	}
	cnt, monitored, err := s.RecordChannelFailure(ctx, ch, "forbidden")
	if err != nil || !monitored || cnt != 1 {
		t.Fatalf("failure: %d %v %v", cnt, monitored, err)
	}
	if _, monitored, _ := s.RecordChannelFailure(ctx, 999, "x"); monitored {
		t.Fatal("unknown channel must not be monitored")
	}
}

func TestClearAndPurge(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	const guild, ch, other snowflake.ID = 1, 10, 20
	if _, err := s.UpsertChannel(ctx, guild, ch, 24*time.Hour, 0); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	a := snowflakeAt(now.Add(-time.Minute), 1)
	b := snowflakeAt(now.Add(-time.Minute), 2)
	c := snowflakeAt(now.Add(-time.Minute), 3)
	if _, err := s.TrackMessages(ctx, []Message{{ID: a, GuildID: guild, ChannelID: ch}}); err != nil {
		t.Fatal(err)
	}
	if n, err := s.ExpediteChannel(ctx, ch); err != nil || n != 1 {
		t.Fatalf("expedite %d %v", n, err)
	}
	// 監視対象外チャンネルでも即時削除予約できる
	if err := s.AddExclusion(ctx, Message{ID: c, GuildID: guild, ChannelID: other}); err != nil {
		t.Fatal(err)
	}
	n, err := s.ScheduleImmediate(ctx, []Message{{ID: b, GuildID: guild, ChannelID: other}, {ID: b, GuildID: guild, ChannelID: other}, {ID: c, GuildID: guild, ChannelID: other}})
	if err != nil || n != 1 {
		t.Fatalf("schedule immediate %d %v", n, err)
	}
	got, _ := s.ClaimDue(ctx, 10, time.Minute, nil)
	if len(got) != 2 {
		t.Fatalf("claimed %+v", got)
	}

	stats, err := s.ChannelStats(ctx, guild)
	if err != nil || stats[ch].Pending != 1 || stats[other].Pending != 1 {
		t.Fatalf("stats %+v %v", stats, err)
	}
	ex, err := s.ListExclusions(ctx, guild, 0)
	if err != nil || len(ex) != 1 {
		t.Fatalf("exclusions %+v %v", ex, err)
	}

	if err := s.UpdateSyncedUntilBatch(ctx, map[snowflake.ID]snowflake.ID{ch: 500}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateSyncedUntil(ctx, ch, 100); err != nil { // 後退しない
		t.Fatal(err)
	}
	if su, ok, _ := s.SyncedUntil(ctx, ch); !ok || su != 500 {
		t.Fatalf("synced_until %v", su)
	}

	if err := s.PurgeGuild(ctx, guild); err != nil {
		t.Fatal(err)
	}
	chs, _ := s.ListChannels(ctx)
	if len(chs) != 0 {
		t.Fatal("guild not purged")
	}
	if ok, _ := s.IsTracked(ctx, b); ok {
		t.Fatal("tracked not purged")
	}
}
