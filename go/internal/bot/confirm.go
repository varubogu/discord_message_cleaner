package bot

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"strings"
	"sync"
	"time"

	disgobot "github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
)

const confirmPrefix = "confirm:"

// confirmAction は確認ダイアログで OK が押されたときの処理。結果メッセージを返す。
type confirmAction func(ctx context.Context) string

type pendingConfirm struct {
	userID snowflake.ID
	appID  snowflake.ID
	token  string
	action confirmAction
	timer  *time.Timer
}

// confirms は確認ダイアログ（Ok / Cancel ボタン）を管理する。
type confirms struct {
	client  *disgobot.Client
	timeout time.Duration
	log     *slog.Logger

	mu      sync.Mutex
	pending map[string]*pendingConfirm
}

func newConfirms(client *disgobot.Client, timeout time.Duration, log *slog.Logger) *confirms {
	return &confirms{client: client, timeout: timeout, log: log, pending: map[string]*pendingConfirm{}}
}

// ask は確認ダイアログを表示する。タイムアウトすると確認メッセージを削除する。
func (c *confirms) ask(e *events.ApplicationCommandInteractionCreate, content string, action confirmAction) {
	id := randomID()
	p := &pendingConfirm{
		userID: e.User().ID,
		appID:  e.ApplicationID(),
		token:  e.Token(),
		action: action,
	}
	c.mu.Lock()
	c.pending[id] = p
	c.mu.Unlock()

	err := e.CreateMessage(discord.NewMessageCreate().
		WithContent(content).
		WithEphemeral(true).
		WithAllowedMentions(&discord.AllowedMentions{}).
		AddActionRow(
			discord.NewPrimaryButton("Ok", confirmPrefix+id+":ok"),
			discord.NewSecondaryButton("Cancel", confirmPrefix+id+":cancel"),
		))
	if err != nil {
		c.take(id)
		c.log.Error("failed to send confirm dialog", slog.Any("err", err))
		return
	}

	p.timer = time.AfterFunc(c.timeout, func() {
		if c.take(id) == nil {
			return
		}
		if err := c.client.Rest.DeleteInteractionResponse(p.appID, p.token); err != nil {
			c.log.Debug("failed to delete timed out confirm dialog", slog.Any("err", err))
		}
	})
}

func (c *confirms) take(id string) *pendingConfirm {
	c.mu.Lock()
	defer c.mu.Unlock()
	p := c.pending[id]
	delete(c.pending, id)
	return p
}

func (c *confirms) peek(id string) *pendingConfirm {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.pending[id]
}

// handle はボタン押下を処理する。確認ダイアログのボタンでなければ false を返す。
func (c *confirms) handle(ctx context.Context, e *events.ComponentInteractionCreate) bool {
	customID := e.Data.CustomID()
	if !strings.HasPrefix(customID, confirmPrefix) {
		return false
	}
	rest := strings.TrimPrefix(customID, confirmPrefix)
	id, choice, _ := strings.Cut(rest, ":")

	if p := c.peek(id); p != nil && p.userID != e.User().ID {
		_ = e.CreateMessage(discord.NewMessageCreate().WithContent("この操作はコマンドを実行した本人のみ可能です。").WithEphemeral(true))
		return true
	}
	p := c.take(id)
	if p == nil {
		_ = e.UpdateMessage(discord.NewMessageUpdate().WithContent("タイムアウトしました。もう一度コマンドを実行してください。").ClearComponents())
		return true
	}
	if p.timer != nil {
		p.timer.Stop()
	}

	if choice != "ok" {
		_ = e.UpdateMessage(discord.NewMessageUpdate().WithContent("キャンセルしました").ClearComponents())
		return true
	}

	// 処理に3秒以上かかってもよいよう、先に応答を保留してから結果で書き換える
	if err := e.DeferUpdateMessage(); err != nil {
		c.log.Error("failed to defer confirm response", slog.Any("err", err))
		return true
	}
	result := p.action(ctx)
	if _, err := c.client.Rest.UpdateInteractionResponse(e.ApplicationID(), e.Token(),
		discord.NewMessageUpdate().WithContent(result).ClearComponents().WithAllowedMentions(&discord.AllowedMentions{})); err != nil {
		c.log.Error("failed to send confirm result", slog.Any("err", err))
	}
	return true
}

func randomID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
