// Package store は PostgreSQL への永続化を担当する。
package store

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Store は DB 操作をまとめたもの。
type Store struct {
	pool *pgxpool.Pool
}

// Open は DB に接続する。
func Open(ctx context.Context, url string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &Store{pool: pool}, nil
}

// Close は接続を閉じる。
func (s *Store) Close() { s.pool.Close() }

// Migrate は未適用のマイグレーションを順に適用する。
func (s *Store) Migrate(ctx context.Context) error {
	files, err := fs.Glob(migrationFS, "migrations/*.sql")
	if err != nil {
		return err
	}
	slices.Sort(files)

	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		// 複数プロセスが同時に起動してもマイグレーションが重複しないようにする
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(727274001)`); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
			version    INTEGER     PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
			return err
		}
		for _, f := range files {
			name := strings.TrimPrefix(f, "migrations/")
			version, err := strconv.Atoi(strings.SplitN(name, "_", 2)[0])
			if err != nil {
				return fmt.Errorf("invalid migration name %q", name)
			}
			var exists bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&exists); err != nil {
				return err
			}
			if exists {
				continue
			}
			sql, err := migrationFS.ReadFile(f)
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, string(sql)); err != nil {
				return fmt.Errorf("apply migration %s: %w", name, err)
			}
			if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
				return err
			}
		}
		return nil
	})
}

// ---------------------------------------------------------------------------
// 監視チャンネル
// ---------------------------------------------------------------------------

// Channel は監視チャンネルの設定。
type Channel struct {
	GuildID      snowflake.ID
	ChannelID    snowflake.ID
	Lifetime     time.Duration
	SyncedUntil  snowflake.ID
	FailureCount int
}

// ListChannels は全監視チャンネルを返す。
func (s *Store) ListChannels(ctx context.Context) ([]Channel, error) {
	rows, err := s.pool.Query(ctx, `SELECT guild_id, channel_id, lifetime_seconds, synced_until, failure_count
		FROM monitored_channels ORDER BY guild_id, channel_id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Channel, error) {
		var g, c, l, su int64
		var fc int
		if err := r.Scan(&g, &c, &l, &su, &fc); err != nil {
			return Channel{}, err
		}
		return Channel{
			GuildID:      snowflake.ID(g),
			ChannelID:    snowflake.ID(c),
			Lifetime:     time.Duration(l) * time.Second,
			SyncedUntil:  snowflake.ID(su),
			FailureCount: fc,
		}, nil
	})
}

// UpsertResult は UpsertChannel の結果。
type UpsertResult struct {
	Created     bool
	Rescheduled int64
}

// UpsertChannel は監視チャンネルを登録・更新する。
//
// 既存チャンネルのライフタイム変更時はメッセージの再取得を行わず、
// 記録済みメッセージの削除予定日時（= 発言日時 + ライフタイム）のみを再計算する。
// syncedUntil は新規登録時のみ使用される（0 なら先頭から同期）。
func (s *Store) UpsertChannel(ctx context.Context, guildID, channelID snowflake.ID, lifetime time.Duration, syncedUntil snowflake.ID) (UpsertResult, error) {
	var res UpsertResult
	secs := int64(lifetime / time.Second)
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `INSERT INTO monitored_channels (guild_id, channel_id, lifetime_seconds, synced_until)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (guild_id, channel_id) DO UPDATE
				SET lifetime_seconds = EXCLUDED.lifetime_seconds,
				    failure_count = 0,
				    updated_at = now()
			RETURNING (xmax = 0)`,
			int64(guildID), int64(channelID), secs, int64(syncedUntil)).Scan(&res.Created); err != nil {
			return err
		}
		if res.Created {
			return nil
		}
		tag, err := tx.Exec(ctx, `UPDATE tracked_messages
			SET delete_at = posted_at + $2 * interval '1 second',
			    next_attempt_at = posted_at + $2 * interval '1 second',
			    attempts = 0
			WHERE channel_id = $1`, int64(channelID), secs)
		if err != nil {
			return err
		}
		res.Rescheduled = tag.RowsAffected()
		return nil
	})
	return res, err
}

// DeleteChannel は監視を解除し、そのチャンネルの削除予定を破棄する。除外設定は残す。
func (s *Store) DeleteChannel(ctx context.Context, guildID, channelID snowflake.ID) (bool, error) {
	var existed bool
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `DELETE FROM monitored_channels WHERE guild_id = $1 AND channel_id = $2`, int64(guildID), int64(channelID))
		if err != nil {
			return err
		}
		existed = tag.RowsAffected() > 0
		_, err = tx.Exec(ctx, `DELETE FROM tracked_messages WHERE channel_id = $1`, int64(channelID))
		return err
	})
	return existed, err
}

// PurgeChannel はチャンネルが消滅した場合に関連データを全て削除する。
func (s *Store) PurgeChannel(ctx context.Context, channelID snowflake.ID) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		for _, q := range []string{
			`DELETE FROM monitored_channels WHERE channel_id = $1`,
			`DELETE FROM tracked_messages WHERE channel_id = $1`,
			`DELETE FROM excluded_messages WHERE channel_id = $1`,
		} {
			if _, err := tx.Exec(ctx, q, int64(channelID)); err != nil {
				return err
			}
		}
		return nil
	})
}

// PurgeGuild は Bot がサーバーから退出した場合に関連データを全て削除する。
func (s *Store) PurgeGuild(ctx context.Context, guildID snowflake.ID) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		for _, q := range []string{
			`DELETE FROM monitored_channels WHERE guild_id = $1`,
			`DELETE FROM tracked_messages WHERE guild_id = $1`,
			`DELETE FROM excluded_messages WHERE guild_id = $1`,
		} {
			if _, err := tx.Exec(ctx, q, int64(guildID)); err != nil {
				return err
			}
		}
		return nil
	})
}

// UpdateSyncedUntil は同期済み位置を進める（後退はしない）。
func (s *Store) UpdateSyncedUntil(ctx context.Context, channelID, messageID snowflake.ID) error {
	_, err := s.pool.Exec(ctx, `UPDATE monitored_channels SET synced_until = GREATEST(synced_until, $2) WHERE channel_id = $1`,
		int64(channelID), int64(messageID))
	return err
}

// UpdateSyncedUntilBatch は複数チャンネルの同期済み位置をまとめて進める。
func (s *Store) UpdateSyncedUntilBatch(ctx context.Context, latest map[snowflake.ID]snowflake.ID) error {
	if len(latest) == 0 {
		return nil
	}
	chs := make([]int64, 0, len(latest))
	ids := make([]int64, 0, len(latest))
	for c, m := range latest {
		chs = append(chs, int64(c))
		ids = append(ids, int64(m))
	}
	_, err := s.pool.Exec(ctx, `UPDATE monitored_channels m SET synced_until = GREATEST(m.synced_until, u.message_id)
		FROM unnest($1::bigint[], $2::bigint[]) AS u(channel_id, message_id)
		WHERE m.channel_id = u.channel_id`, chs, ids)
	return err
}

// RecordChannelFailure はチャンネル単位の失敗を記録し、連続失敗回数を返す。監視対象外なら monitored=false。
func (s *Store) RecordChannelFailure(ctx context.Context, channelID snowflake.ID, reason string) (count int, monitored bool, err error) {
	err = s.pool.QueryRow(ctx, `UPDATE monitored_channels
		SET failure_count = failure_count + 1, last_failure_at = now(), last_failure_reason = $2
		WHERE channel_id = $1 RETURNING failure_count`, int64(channelID), reason).Scan(&count)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	return count, err == nil, err
}

// ResetChannelFailure は連続失敗回数をリセットする。
func (s *Store) ResetChannelFailure(ctx context.Context, channelID snowflake.ID) error {
	_, err := s.pool.Exec(ctx, `UPDATE monitored_channels SET failure_count = 0 WHERE channel_id = $1 AND failure_count <> 0`, int64(channelID))
	return err
}

// ---------------------------------------------------------------------------
// 削除予定メッセージ
// ---------------------------------------------------------------------------

// Message はメッセージの識別子。発言日時は Snowflake から算出する。
type Message struct {
	ID        snowflake.ID
	GuildID   snowflake.ID
	ChannelID snowflake.ID
}

// TrackMessages は監視チャンネルのメッセージを記録する。
// 削除予定日時は 発言日時 + チャンネルのライフタイム。監視対象外・除外設定済み・記録済みのものは無視する。
func (s *Store) TrackMessages(ctx context.Context, msgs []Message) (int64, error) {
	if len(msgs) == 0 {
		return 0, nil
	}
	ids, chs, posted := splitMessages(msgs)
	tag, err := s.pool.Exec(ctx, `INSERT INTO tracked_messages (message_id, guild_id, channel_id, posted_at, delete_at, next_attempt_at)
		SELECT u.message_id, m.guild_id, u.channel_id, u.posted_at,
		       u.posted_at + m.lifetime_seconds * interval '1 second',
		       u.posted_at + m.lifetime_seconds * interval '1 second'
		FROM unnest($1::bigint[], $2::bigint[], $3::timestamptz[]) AS u(message_id, channel_id, posted_at)
		JOIN monitored_channels m ON m.channel_id = u.channel_id
		WHERE NOT EXISTS (SELECT 1 FROM excluded_messages e WHERE e.message_id = u.message_id)
		ON CONFLICT (message_id) DO NOTHING`, ids, chs, posted)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ScheduleImmediate はメッセージを即時削除予定として登録する（/clear 用。監視対象外のチャンネルでも可）。
func (s *Store) ScheduleImmediate(ctx context.Context, msgs []Message) (int64, error) {
	if len(msgs) == 0 {
		return 0, nil
	}
	msgs = dedupe(msgs)
	ids, chs, posted := splitMessages(msgs)
	gs := make([]int64, len(msgs))
	for i, m := range msgs {
		gs[i] = int64(m.GuildID)
	}
	tag, err := s.pool.Exec(ctx, `INSERT INTO tracked_messages (message_id, guild_id, channel_id, posted_at, delete_at, next_attempt_at)
		SELECT u.message_id, u.guild_id, u.channel_id, u.posted_at, now(), now()
		FROM unnest($1::bigint[], $2::bigint[], $3::timestamptz[], $4::bigint[]) AS u(message_id, channel_id, posted_at, guild_id)
		WHERE NOT EXISTS (SELECT 1 FROM excluded_messages e WHERE e.message_id = u.message_id)
		ON CONFLICT (message_id) DO UPDATE SET delete_at = now(), next_attempt_at = now(), attempts = 0`, ids, chs, posted, gs)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ExpediteChannel はチャンネル内の記録済みメッセージを全て即時削除予定にする（/clear 用）。
func (s *Store) ExpediteChannel(ctx context.Context, channelID snowflake.ID) (int64, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE tracked_messages SET delete_at = now(), next_attempt_at = now(), attempts = 0
		WHERE channel_id = $1 AND delete_at > now()`, int64(channelID))
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// RemoveTracked は削除済み（または削除不要になった）メッセージの記録を消す。
func (s *Store) RemoveTracked(ctx context.Context, ids []snowflake.ID) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM tracked_messages WHERE message_id = ANY($1::bigint[])`, toInt64s(ids))
	return err
}

// Claimed はリースを取得した削除対象メッセージ。
type Claimed struct {
	MessageID snowflake.ID
	GuildID   snowflake.ID
	ChannelID snowflake.ID
	PostedAt  time.Time
	DeleteAt  time.Time
	Attempts  int
}

// ClaimDue は削除予定日時を過ぎたメッセージを最大 limit 件取得し、lease の間は再取得されないようにする。
// excludeChannels のチャンネル（メモリ内キューが一杯）は対象外にする。
func (s *Store) ClaimDue(ctx context.Context, limit int, lease time.Duration, excludeChannels []snowflake.ID) ([]Claimed, error) {
	rows, err := s.pool.Query(ctx, `WITH due AS (
			SELECT message_id FROM tracked_messages
			WHERE next_attempt_at <= now() AND channel_id <> ALL($3::bigint[])
			ORDER BY next_attempt_at
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE tracked_messages t SET next_attempt_at = now() + $2 * interval '1 millisecond'
		FROM due WHERE t.message_id = due.message_id
		RETURNING t.message_id, t.guild_id, t.channel_id, t.posted_at, t.delete_at, t.attempts`,
		limit, lease.Milliseconds(), toInt64s(excludeChannels))
	if err != nil {
		return nil, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Claimed, error) {
		var c Claimed
		var id, g, ch int64
		if err := r.Scan(&id, &g, &ch, &c.PostedAt, &c.DeleteAt, &c.Attempts); err != nil {
			return c, err
		}
		c.MessageID, c.GuildID, c.ChannelID = snowflake.ID(id), snowflake.ID(g), snowflake.ID(ch)
		return c, nil
	})
	if err != nil {
		return nil, err
	}
	slices.SortFunc(items, func(a, b Claimed) int { return a.DeleteAt.Compare(b.DeleteAt) })
	return items, nil
}

// RetryLater は一時的な失敗をしたメッセージを retryAt に再試行させる。
// 試行回数が maxAttempts に達したものは記録を削除し、その件数を返す。
func (s *Store) RetryLater(ctx context.Context, ids []snowflake.ID, retryAt time.Time, reason string, maxAttempts int) (dropped int64, err error) {
	if len(ids) == 0 {
		return 0, nil
	}
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE tracked_messages SET attempts = attempts + 1, next_attempt_at = $2, last_error = $3
			WHERE message_id = ANY($1::bigint[])`, toInt64s(ids), retryAt, reason); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `DELETE FROM tracked_messages WHERE message_id = ANY($1::bigint[]) AND attempts >= $2`, toInt64s(ids), maxAttempts)
		if err != nil {
			return err
		}
		dropped = tag.RowsAffected()
		return nil
	})
	return dropped, err
}

// BackoffChannel はチャンネル内の全削除予定を until まで保留する（権限エラー時など無駄なリクエストを避ける）。
func (s *Store) BackoffChannel(ctx context.Context, channelID snowflake.ID, until time.Time, reason string) error {
	_, err := s.pool.Exec(ctx, `UPDATE tracked_messages SET next_attempt_at = GREATEST(next_attempt_at, $2), last_error = $3
		WHERE channel_id = $1`, int64(channelID), until, reason)
	return err
}

// DropChannelTracked はチャンネル内の削除予定を全て破棄する（監視対象外チャンネルの /clear が失敗した場合など）。
func (s *Store) DropChannelTracked(ctx context.Context, channelID snowflake.ID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM tracked_messages WHERE channel_id = $1`, int64(channelID))
	return err
}

// ReleaseClaims は終了時に未処理のリースを解放し、再起動後すぐに処理されるようにする。
func (s *Store) ReleaseClaims(ctx context.Context, ids []snowflake.ID) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := s.pool.Exec(ctx, `UPDATE tracked_messages SET next_attempt_at = LEAST(next_attempt_at, now())
		WHERE message_id = ANY($1::bigint[])`, toInt64s(ids))
	return err
}

// ChannelStat はチャンネルごとの削除予定の統計。
type ChannelStat struct {
	Pending    int64
	NextDelete *time.Time
}

// ChannelStats はサーバー内チャンネルごとの削除予定件数と次回削除予定日時を返す。
func (s *Store) ChannelStats(ctx context.Context, guildID snowflake.ID) (map[snowflake.ID]ChannelStat, error) {
	rows, err := s.pool.Query(ctx, `SELECT channel_id, count(*), min(delete_at) FROM tracked_messages
		WHERE guild_id = $1 GROUP BY channel_id`, int64(guildID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	res := map[snowflake.ID]ChannelStat{}
	for rows.Next() {
		var ch, n int64
		var next time.Time
		if err := rows.Scan(&ch, &n, &next); err != nil {
			return nil, err
		}
		res[snowflake.ID(ch)] = ChannelStat{Pending: n, NextDelete: &next}
	}
	return res, rows.Err()
}

// IsTracked はメッセージが記録済みか返す。
func (s *Store) IsTracked(ctx context.Context, messageID snowflake.ID) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM tracked_messages WHERE message_id = $1)`, int64(messageID)).Scan(&ok)
	return ok, err
}

// ---------------------------------------------------------------------------
// 除外設定
// ---------------------------------------------------------------------------

// AddExclusion は除外設定を追加し、削除予定から外す。
func (s *Store) AddExclusion(ctx context.Context, m Message) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO excluded_messages (message_id, guild_id, channel_id) VALUES ($1, $2, $3)
			ON CONFLICT (message_id) DO NOTHING`, int64(m.ID), int64(m.GuildID), int64(m.ChannelID)); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `DELETE FROM tracked_messages WHERE message_id = $1`, int64(m.ID))
		return err
	})
}

// RemoveExclusion は除外設定を解除する。チャンネルが監視対象なら、発言日時から削除予定日時を再計算して記録し直す。
func (s *Store) RemoveExclusion(ctx context.Context, m Message) (existed bool, err error) {
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `DELETE FROM excluded_messages WHERE message_id = $1 AND guild_id = $2 AND channel_id = $3`,
			int64(m.ID), int64(m.GuildID), int64(m.ChannelID))
		if err != nil {
			return err
		}
		existed = tag.RowsAffected() > 0
		if !existed {
			return nil
		}
		posted := m.ID.Time()
		_, err = tx.Exec(ctx, `INSERT INTO tracked_messages (message_id, guild_id, channel_id, posted_at, delete_at, next_attempt_at)
			SELECT $1, m.guild_id, m.channel_id, $3::timestamptz,
			       $3::timestamptz + m.lifetime_seconds * interval '1 second',
			       $3::timestamptz + m.lifetime_seconds * interval '1 second'
			FROM monitored_channels m WHERE m.channel_id = $2
			ON CONFLICT (message_id) DO NOTHING`, int64(m.ID), int64(m.ChannelID), posted)
		return err
	})
	return existed, err
}

// ListExclusions はサーバー内（channelID が 0 以外ならそのチャンネル内）の除外設定を返す。
func (s *Store) ListExclusions(ctx context.Context, guildID, channelID snowflake.ID) ([]Message, error) {
	rows, err := s.pool.Query(ctx, `SELECT message_id, guild_id, channel_id FROM excluded_messages
		WHERE guild_id = $1 AND ($2 = 0 OR channel_id = $2) ORDER BY channel_id, message_id`, int64(guildID), int64(channelID))
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Message, error) {
		var id, g, c int64
		err := r.Scan(&id, &g, &c)
		return Message{ID: snowflake.ID(id), GuildID: snowflake.ID(g), ChannelID: snowflake.ID(c)}, err
	})
}

// ---------------------------------------------------------------------------

func splitMessages(msgs []Message) (ids, chs []int64, posted []time.Time) {
	ids = make([]int64, len(msgs))
	chs = make([]int64, len(msgs))
	posted = make([]time.Time, len(msgs))
	for i, m := range msgs {
		ids[i] = int64(m.ID)
		chs[i] = int64(m.ChannelID)
		posted[i] = m.ID.Time()
	}
	return
}

func dedupe(msgs []Message) []Message {
	seen := make(map[snowflake.ID]struct{}, len(msgs))
	out := msgs[:0:0]
	for _, m := range msgs {
		if _, ok := seen[m.ID]; ok {
			continue
		}
		seen[m.ID] = struct{}{}
		out = append(out, m)
	}
	return out
}

func toInt64s(ids []snowflake.ID) []int64 {
	out := make([]int64, len(ids))
	for i, id := range ids {
		out[i] = int64(id)
	}
	return out
}

// SyncedUntil はチャンネルの同期済み位置を返す。監視対象外なら ok=false。
func (s *Store) SyncedUntil(ctx context.Context, channelID snowflake.ID) (id snowflake.ID, ok bool, err error) {
	var v int64
	err = s.pool.QueryRow(ctx, `SELECT synced_until FROM monitored_channels WHERE channel_id = $1`, int64(channelID)).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return snowflake.ID(v), true, nil
}
