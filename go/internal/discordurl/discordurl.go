// Package discordurl はメッセージURL・メッセージIDの解析を行う。
package discordurl

import (
	"errors"
	"regexp"
	"strings"

	"github.com/disgoorg/snowflake/v2"
)

// MessageRef はメッセージURLから得られる識別子の組。
type MessageRef struct {
	GuildID   snowflake.ID
	ChannelID snowflake.ID
	MessageID snowflake.ID
}

var messageURL = regexp.MustCompile(`^https://(?:(?:ptb|canary)\.)?discord(?:app)?\.com/channels/(\d+)/(\d+)/(\d+)`)

// ParseMessageURL はメッセージURLを解析する。
func ParseMessageURL(s string) (MessageRef, error) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "https://") {
		return MessageRef{}, errors.New("メッセージURLがURL形式ではありません")
	}
	m := messageURL.FindStringSubmatch(s)
	if m == nil {
		return MessageRef{}, errors.New("URLの解析に失敗しました。")
	}
	var ref MessageRef
	var err error
	if ref.GuildID, err = snowflake.Parse(m[1]); err != nil {
		return MessageRef{}, errors.New("URLの解析に失敗しました。")
	}
	if ref.ChannelID, err = snowflake.Parse(m[2]); err != nil {
		return MessageRef{}, errors.New("URLの解析に失敗しました。")
	}
	if ref.MessageID, err = snowflake.Parse(m[3]); err != nil {
		return MessageRef{}, errors.New("URLの解析に失敗しました。")
	}
	return ref, nil
}
