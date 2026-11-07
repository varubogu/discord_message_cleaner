// Package i18n は messages.json（Python 版と同一形式）からメッセージを引く。
package i18n

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

// Code は失敗理由コード。Python 版 FailedReasonCode と同じ値。
type Code string

const (
	Unknown                      Code = "E0000"
	InvalidToken                 Code = "E0001"
	RateLimit                    Code = "E0002"
	GuildNotFound                Code = "E0003"
	GuildAccessDenied            Code = "E0004"
	ChannelNotFound              Code = "E0005"
	ChannelAccessDenied          Code = "E0006"
	MessageDeletePermDenied      Code = "E0007"
	MessageReadPermDenied        Code = "E0008"
	MessageNotFound              Code = "E0009"
	MessageAccessDenied          Code = "E0010"
	MessageReadHistoryPermDenied Code = "E0011"
	NoBotUsagePermission         Code = "E0012"
	ThreadArchived               Code = "E0013"
	MessageDeleteDenied          Code = "E0014"
)

//go:embed messages.json
var raw []byte

type entry struct {
	Ja string `json:"ja"`
	En string `json:"en"`
}

// Messages はメッセージ辞書。
type Messages struct {
	m           map[string]entry
	displayLang string
	logLang     string
}

// Load は埋め込まれた messages.json を読み込む。
func Load(displayLang, logLang string) (*Messages, error) {
	var m map[string]entry
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parse messages.json: %w", err)
	}
	return &Messages{m: m, displayLang: displayLang, logLang: logLang}, nil
}

// Lang は表示言語を返す。
func (ms *Messages) Lang() string { return ms.displayLang }

func (ms *Messages) get(key, lang string) string {
	e, ok := ms.m[key]
	if !ok {
		return key
	}
	if lang == "ja" {
		return e.Ja
	}
	return e.En
}

// Get は表示言語でメッセージを返す。
func (ms *Messages) Get(key string) string { return ms.get(key, ms.displayLang) }

// Display はエラーコードの表示用メッセージを返す。
func (ms *Messages) Display(c Code) string { return ms.get(string(c), ms.displayLang) }

// Log はエラーコードのログ用メッセージを返す。
func (ms *Messages) Log(c Code) string { return ms.get(string(c), ms.logLang) }
