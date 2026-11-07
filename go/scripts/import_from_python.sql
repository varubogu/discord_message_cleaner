-- Python版の設定を Go 版のテーブルへ取り込む。
-- Python版と同じDBで Go 版を一度起動（テーブル作成）した後に実行する。
--   psql "$DATABASE_URL" -f scripts/import_from_python.sql
--
-- 取り込んだチャンネルは synced_until = 0 のため、次回起動時に履歴を先頭から同期して
-- 既存メッセージも削除予定として登録する（SYNC_MAX_PAGES で上限を設定可能）。

BEGIN;

INSERT INTO monitored_channels (guild_id, channel_id, lifetime_seconds)
SELECT guild_id, channel_id, GREATEST(1, EXTRACT(EPOCH FROM "interval")::bigint)
FROM monitoring_channels
WHERE "interval" IS NOT NULL
ON CONFLICT DO NOTHING;

INSERT INTO excluded_messages (message_id, guild_id, channel_id)
SELECT message_id, guild_id, channel_id
FROM exclusion_message
ON CONFLICT DO NOTHING;

COMMIT;
