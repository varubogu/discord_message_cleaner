-- 削除監視対象のチャンネル
CREATE TABLE IF NOT EXISTS monitored_channels (
    guild_id            BIGINT      NOT NULL,
    channel_id          BIGINT      NOT NULL,
    lifetime_seconds    BIGINT      NOT NULL CHECK (lifetime_seconds > 0),
    -- このメッセージIDまでは履歴をDBに取り込み済み（0 = 先頭から未同期）
    synced_until        BIGINT      NOT NULL DEFAULT 0,
    -- 削除時にチャンネル単位のエラー（権限不足など）が連続した回数
    failure_count       INTEGER     NOT NULL DEFAULT 0,
    last_failure_at     TIMESTAMPTZ,
    last_failure_reason TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (guild_id, channel_id)
);
COMMENT ON TABLE monitored_channels IS '削除監視対象のチャンネル一覧';
CREATE UNIQUE INDEX IF NOT EXISTS ux_monitored_channels_channel ON monitored_channels (channel_id);

-- 削除対象から除外するメッセージ
CREATE TABLE IF NOT EXISTS excluded_messages (
    message_id BIGINT      PRIMARY KEY,
    guild_id   BIGINT      NOT NULL,
    channel_id BIGINT      NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
COMMENT ON TABLE excluded_messages IS '削除対象から除外するメッセージ一覧';
CREATE INDEX IF NOT EXISTS ix_excluded_messages_guild_channel ON excluded_messages (guild_id, channel_id);

-- 削除予定のメッセージ（メッセージ投稿イベントで記録する）
CREATE TABLE IF NOT EXISTS tracked_messages (
    message_id      BIGINT      PRIMARY KEY,
    guild_id        BIGINT      NOT NULL,
    channel_id      BIGINT      NOT NULL,
    -- 発言日時
    posted_at       TIMESTAMPTZ NOT NULL,
    -- 削除予定日時
    delete_at       TIMESTAMPTZ NOT NULL,
    -- 次に削除処理を行ってよい日時（= 削除予定日時。処理中はリース期限、失敗時はリトライ日時）
    next_attempt_at TIMESTAMPTZ NOT NULL,
    attempts        INTEGER     NOT NULL DEFAULT 0,
    last_error      TEXT
);
COMMENT ON TABLE tracked_messages IS '削除予定のメッセージ一覧';
CREATE INDEX IF NOT EXISTS ix_tracked_messages_next_attempt ON tracked_messages (next_attempt_at);
CREATE INDEX IF NOT EXISTS ix_tracked_messages_channel ON tracked_messages (channel_id);
