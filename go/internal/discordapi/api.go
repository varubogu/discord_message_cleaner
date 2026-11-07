// Package discordapi は Bot が自発的に送る Discord REST リクエストを1箇所に集約する。
//
// レートリミット対策は2段構え。
//
//  1. ここで持つトークンバケット（golang.org/x/time/rate）で Bot 全体の送信量を
//     グローバル上限（50 req/s）より低く抑え、事前に待つ。
//  2. disgo の rest.RateLimiter がレスポンスヘッダ（X-RateLimit-*）を見てルート/バケット単位に待機し、
//     万一 429 を受けた場合は Retry-After 後に再送する。
//
// インタラクション応答（コマンドへの返信）はグローバル上限の対象外のため、ここを経由しない。
package discordapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"golang.org/x/time/rate"
)

// MaxBulkDelete は一括削除 API で1回に指定できる最大件数。
const MaxBulkDelete = 100

// MinBulkDelete は一括削除 API で1回に指定できる最小件数。
const MinBulkDelete = 2

// API はレート制御付きの Discord REST クライアント。
type API struct {
	rest    rest.Rest
	limiter *rate.Limiter
}

// New は API を作成する。
func New(r rest.Rest, perSecond float64, burst int) *API {
	return &API{rest: r, limiter: rate.NewLimiter(rate.Limit(perSecond), burst)}
}

func (a *API) wait(ctx context.Context) error {
	return a.limiter.Wait(ctx)
}

// DeleteMessage はメッセージを1件削除する。
func (a *API) DeleteMessage(ctx context.Context, channelID, messageID snowflake.ID) error {
	if err := a.wait(ctx); err != nil {
		return err
	}
	return a.rest.DeleteMessage(channelID, messageID, rest.WithCtx(ctx))
}

// BulkDeleteMessages は2〜100件のメッセージを1リクエストで削除する（投稿から14日以内のもののみ）。
func (a *API) BulkDeleteMessages(ctx context.Context, channelID snowflake.ID, messageIDs []snowflake.ID) error {
	if err := a.wait(ctx); err != nil {
		return err
	}
	return a.rest.BulkDeleteMessages(channelID, messageIDs, rest.WithCtx(ctx))
}

// GetMessagesAfter は after より新しいメッセージを最大 limit 件取得する。
func (a *API) GetMessagesAfter(ctx context.Context, channelID, after snowflake.ID, limit int) ([]discord.Message, error) {
	if err := a.wait(ctx); err != nil {
		return nil, err
	}
	// disgo は 0 のパラメータを送らない（= 最新から取得になる）ため、先頭から取得する場合は 1 を指定する
	return a.rest.GetMessages(channelID, 0, 0, max(after, 1), limit, rest.WithCtx(ctx))
}

// GetMessagesBefore は before より古いメッセージを最大 limit 件取得する（before が 0 なら最新から）。
func (a *API) GetMessagesBefore(ctx context.Context, channelID, before snowflake.ID, limit int) ([]discord.Message, error) {
	if err := a.wait(ctx); err != nil {
		return nil, err
	}
	return a.rest.GetMessages(channelID, 0, before, 0, limit, rest.WithCtx(ctx))
}

// GetMessage はメッセージを1件取得する。
func (a *API) GetMessage(ctx context.Context, channelID, messageID snowflake.ID) (*discord.Message, error) {
	if err := a.wait(ctx); err != nil {
		return nil, err
	}
	return a.rest.GetMessage(channelID, messageID, rest.WithCtx(ctx))
}

// GetChannel はチャンネルを取得する。
func (a *API) GetChannel(ctx context.Context, channelID snowflake.ID) (discord.Channel, error) {
	if err := a.wait(ctx); err != nil {
		return nil, err
	}
	return a.rest.GetChannel(channelID, rest.WithCtx(ctx))
}

// ErrorKind は Discord API エラーの分類。
type ErrorKind int

const (
	// KindTemporary は時間をおいて再試行すべきエラー（5xx、429、ネットワーク断など）。
	KindTemporary ErrorKind = iota
	// KindMessageGone はメッセージが既に存在しない（削除済みとして扱ってよい）。
	KindMessageGone
	// KindUndeletable はシステムメッセージなど削除できないメッセージ。
	KindUndeletable
	// KindChannelGone はチャンネルが存在しない。
	KindChannelGone
	// KindChannelDenied は権限不足・アーカイブ済みスレッドなど、チャンネル単位で削除できない状態。
	KindChannelDenied
	// KindBadBulk は一括削除の条件（14日以内など）を満たしていない。個別削除に切り替える。
	KindBadBulk
	// KindCanceled はシャットダウンなどによるキャンセル。
	KindCanceled
)

// Classify はエラーを分類する。
func Classify(err error) ErrorKind {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return KindCanceled
	}
	var re *rest.Error
	if !errors.As(err, &re) {
		return KindTemporary
	}
	switch re.Code {
	case rest.JSONErrorCodeUnknownMessage:
		return KindMessageGone
	case rest.JSONErrorCodeCannotExecuteActionOnSystemMessage:
		return KindUndeletable
	case rest.JSONErrorCodeUnknownChannel, rest.JSONErrorCodeUnknownGuild:
		return KindChannelGone
	case rest.JSONErrorCodeMissingAccess, rest.JSONErrorCodeLackPermissionsToPerformAction, rest.JSONErrorCodeOperationOnArchivedThread:
		return KindChannelDenied
	case rest.JSONErrorCodeMessageTooOldToBulkDelete, rest.JSONErrorCodeInvalidFormBody:
		return KindBadBulk
	}
	if re.Response != nil {
		switch re.Response.StatusCode {
		case http.StatusNotFound:
			// コードが付かない 404 はチャンネル消滅と断定せず、メッセージ単位で破棄する
			return KindMessageGone
		case http.StatusForbidden:
			return KindChannelDenied
		case http.StatusBadRequest:
			return KindBadBulk
		}
	}
	return KindTemporary
}
