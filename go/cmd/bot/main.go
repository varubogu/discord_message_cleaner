// Command bot は Discord メッセージクリーナー（Go 版）を起動する。
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/varubogu/discord_message_cleaner_go/internal/bot"
	"github.com/varubogu/discord_message_cleaner_go/internal/config"
	"github.com/varubogu/discord_message_cleaner_go/internal/i18n"
	"github.com/varubogu/discord_message_cleaner_go/internal/registry"
	"github.com/varubogu/discord_message_cleaner_go/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", slog.Any("err", err))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	msgs, err := i18n.Load(cfg.MessageLanguage, cfg.LogLanguage)
	if err != nil {
		return err
	}

	st, err := openStore(ctx, cfg.DatabaseURL, logger)
	if err != nil {
		return err
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		return err
	}

	reg := registry.New()
	channels, err := st.ListChannels(ctx)
	if err != nil {
		return err
	}
	for _, c := range channels {
		reg.Set(registry.Entry{GuildID: c.GuildID, ChannelID: c.ChannelID, Lifetime: c.Lifetime, FailureCount: c.FailureCount})
	}
	logger.Info("loaded monitored channels", slog.Int("count", len(channels)))

	b, err := bot.New(bot.Deps{Config: cfg, Store: st, Registry: reg, Messages: msgs, Logger: logger})
	if err != nil {
		return err
	}
	return b.Run(ctx)
}

// openStore は DB の起動を待ちながら接続する（docker compose で同時に起動した場合など）。
func openStore(ctx context.Context, url string, logger *slog.Logger) (*store.Store, error) {
	var lastErr error
	for i := range 30 {
		st, err := store.Open(ctx, url)
		if err == nil {
			return st, nil
		}
		lastErr = err
		logger.Warn("waiting for database", slog.Int("attempt", i+1), slog.Any("err", err))
		select {
		case <-time.After(2 * time.Second):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, lastErr
}
