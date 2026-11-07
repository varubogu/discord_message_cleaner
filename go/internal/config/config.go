// Package config は環境変数から Bot の設定値を読み込む。
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/snowflake/v2"
)

// Config は Bot 全体の設定値。
type Config struct {
	DiscordToken string
	// DevGuildID が設定されている場合、スラッシュコマンドをそのギルドにのみ登録する（即時反映される開発用）。
	DevGuildID snowflake.ID

	DatabaseURL string

	MessageLanguage string
	LogLanguage     string
	LogLevel        slog.Level
	AdminRoleName   string

	// --- 削除スケジューラ ---

	// SchedulerInterval は DB に削除期限切れのメッセージがあるか確認する周期。
	SchedulerInterval time.Duration
	// ClaimBatchSize は1周期で DB から取得（リース）する最大件数。
	ClaimBatchSize int
	// ClaimLease はリースの有効期間。期限内に処理されなければ再取得される。
	ClaimLease time.Duration
	// ChannelQueueLimit はチャンネルごとのメモリ内キューの上限。超えたチャンネルは次周期の取得対象から外す。
	ChannelQueueLimit int
	// DeleteWorkers は同時にリクエストを投げるチャンネルワーカー数の上限。
	DeleteWorkers int
	// MaxDeleteAttempts はメッセージ単位の一時エラー時の最大試行回数。
	MaxDeleteAttempts int
	// ChannelFailureLimit はチャンネル単位の権限エラー等がこの回数連続したら監視を解除する。
	ChannelFailureLimit int
	// ChannelFailureBackoff はチャンネル単位のエラー時、そのチャンネルの削除を保留する時間。
	ChannelFailureBackoff time.Duration

	// --- Discord API ---

	// APIRatePerSecond は Bot 全体で1秒あたりに送る REST リクエスト数の上限（Discordのグローバル上限は50/s）。
	APIRatePerSecond float64
	// APIBurst はトークンバケットのバースト値。
	APIBurst int

	// --- イベント記録 ---

	TrackerFlushInterval time.Duration
	TrackerBatchSize     int
	TrackerQueueSize     int

	// --- 履歴同期 ---

	// BackfillOnEnable が true の場合、新規に監視を開始したチャンネルの既存メッセージを1度だけ取得して DB に登録する。
	BackfillOnEnable bool
	// SyncMaxPages は1回の同期でチャンネルごとに取得する最大ページ数（1ページ100件）。0 は無制限。
	SyncMaxPages int
	// SyncConcurrency は同時に同期するチャンネル数。
	SyncConcurrency int

	// ClearMaxMessages は /clear で1回に削除予約する最大件数。
	ClearMaxMessages int

	// ConfirmTimeout は確認ダイアログのタイムアウト。
	ConfirmTimeout time.Duration
}

// Load は環境変数から設定を読み込む。
func Load() (*Config, error) {
	var errs []error
	c := &Config{}

	c.DiscordToken = os.Getenv("DISCORD_TOKEN")
	if c.DiscordToken == "" {
		errs = append(errs, errors.New("DISCORD_TOKEN is required"))
	}
	if v := os.Getenv("DEV_GUILD_ID"); v != "" {
		id, err := snowflake.Parse(v)
		if err != nil {
			errs = append(errs, fmt.Errorf("DEV_GUILD_ID: %w", err))
		}
		c.DevGuildID = id
	}

	dbURL, err := databaseURL()
	if err != nil {
		errs = append(errs, err)
	}
	c.DatabaseURL = dbURL

	c.MessageLanguage = lang(os.Getenv("MESSAGE_LANGUAGE"))
	c.LogLanguage = lang(os.Getenv("LOG_LANGUAGE"))
	c.LogLevel = logLevel(os.Getenv("LOG_LEVEL"))
	c.AdminRoleName = str("ADMIN_ROLE_NAME", "message_cleaner_admin")

	c.SchedulerInterval = dur("SCHEDULER_INTERVAL", time.Second)
	c.ClaimBatchSize = num("CLAIM_BATCH_SIZE", 1000)
	c.ClaimLease = dur("CLAIM_LEASE", 2*time.Minute)
	c.ChannelQueueLimit = num("CHANNEL_QUEUE_LIMIT", 500)
	c.DeleteWorkers = num("DELETE_WORKERS", 8)
	c.MaxDeleteAttempts = num("MAX_DELETE_ATTEMPTS", 10)
	// Python 版の MESSAGE_DELETE_FAILURES と同じ意味
	c.ChannelFailureLimit = num("CHANNEL_FAILURE_LIMIT", num("MESSAGE_DELETE_FAILURES", 3))
	c.ChannelFailureBackoff = dur("CHANNEL_FAILURE_BACKOFF", 10*time.Minute)

	c.APIRatePerSecond = float("API_RATE_PER_SECOND", 40)
	c.APIBurst = num("API_BURST", 10)

	c.TrackerFlushInterval = dur("TRACKER_FLUSH_INTERVAL", 250*time.Millisecond)
	c.TrackerBatchSize = num("TRACKER_BATCH_SIZE", 500)
	c.TrackerQueueSize = num("TRACKER_QUEUE_SIZE", 20000)

	c.BackfillOnEnable = boolean("BACKFILL_ON_ENABLE", true)
	c.SyncMaxPages = num("SYNC_MAX_PAGES", 0)
	c.SyncConcurrency = num("SYNC_CONCURRENCY", 2)

	c.ClearMaxMessages = num("CLEAR_MAX_MESSAGES", 1000)
	c.ConfirmTimeout = dur("CONFIRM_TIMEOUT", 10*time.Second)

	if c.ClaimBatchSize <= 0 || c.ChannelQueueLimit <= 0 || c.DeleteWorkers <= 0 || c.SyncConcurrency <= 0 {
		errs = append(errs, errors.New("CLAIM_BATCH_SIZE, CHANNEL_QUEUE_LIMIT, DELETE_WORKERS, SYNC_CONCURRENCY must be > 0"))
	}
	if c.APIRatePerSecond <= 0 || c.APIBurst <= 0 {
		errs = append(errs, errors.New("API_RATE_PER_SECOND and API_BURST must be > 0"))
	}
	if c.SchedulerInterval <= 0 || c.TrackerFlushInterval <= 0 {
		errs = append(errs, errors.New("SCHEDULER_INTERVAL and TRACKER_FLUSH_INTERVAL must be > 0"))
	}

	return c, errors.Join(errs...)
}

// databaseURL は DATABASE_URL、なければ Python 版互換の DBHOST 等から接続文字列を組み立てる。
func databaseURL() (string, error) {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v, nil
	}
	host := os.Getenv("DBHOST")
	user := os.Getenv("DBUSER")
	db := os.Getenv("DBDATABASE")
	if host == "" || user == "" || db == "" {
		return "", errors.New("DATABASE_URL or DBHOST/DBUSER/DBPASSWORD/DBDATABASE is required")
	}
	port := str("DBPORT", "5432")
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, os.Getenv("DBPASSWORD")),
		Host:   host + ":" + port,
		Path:   "/" + db,
	}
	q := u.Query()
	q.Set("sslmode", str("DBSSLMODE", "disable"))
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func lang(v string) string {
	switch strings.ToLower(v) {
	case "ja":
		return "ja"
	default:
		return "en"
	}
}

func logLevel(v string) slog.Level {
	switch strings.ToLower(v) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func str(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func num(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		slog.Warn("invalid int env, use default", slog.String("key", key), slog.String("value", v), slog.Int("default", def))
		return def
	}
	return n
}

func float(key string, def float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		slog.Warn("invalid float env, use default", slog.String("key", key), slog.String("value", v), slog.Float64("default", def))
		return def
	}
	return n
}

func dur(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		// 単位なしの数値は秒として扱う
		if n, err2 := strconv.ParseFloat(v, 64); err2 == nil {
			return time.Duration(n * float64(time.Second))
		}
		slog.Warn("invalid duration env, use default", slog.String("key", key), slog.String("value", v), slog.Duration("default", def))
		return def
	}
	return d
}

func boolean(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		slog.Warn("invalid bool env, use default", slog.String("key", key), slog.String("value", v), slog.Bool("default", def))
		return def
	}
	return b
}
