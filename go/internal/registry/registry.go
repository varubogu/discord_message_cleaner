// Package registry は監視チャンネル設定をメモリ上に保持する。
//
// メッセージ投稿イベントは大量に届くため、監視対象かどうかの判定で DB へ問い合わせないようにする。
// DB を正とし、起動時と設定変更時に同期する（単一プロセスでの稼働を前提とする）。
package registry

import (
	"sync"
	"time"

	"github.com/disgoorg/snowflake/v2"
)

// Entry は監視チャンネル1件分の設定。
type Entry struct {
	GuildID      snowflake.ID
	ChannelID    snowflake.ID
	Lifetime     time.Duration
	FailureCount int
	// Syncing は履歴の同期中であることを示す。同期中は投稿イベントで同期済み位置を進めない。
	Syncing bool
}

// Registry は監視チャンネルの集合。
type Registry struct {
	mu       sync.RWMutex
	channels map[snowflake.ID]Entry
}

// New は空の Registry を返す。
func New() *Registry {
	return &Registry{channels: map[snowflake.ID]Entry{}}
}

// Get はチャンネルの設定を返す。
func (r *Registry) Get(channelID snowflake.ID) (Entry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.channels[channelID]
	return e, ok
}

// IsMonitored はチャンネルが監視対象か返す。
func (r *Registry) IsMonitored(channelID snowflake.ID) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.channels[channelID]
	return ok
}

// Set はチャンネルの設定を登録・更新する。
func (r *Registry) Set(e Entry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if old, ok := r.channels[e.ChannelID]; ok {
		e.Syncing = e.Syncing || old.Syncing
	}
	r.channels[e.ChannelID] = e
}

// Update はチャンネルの設定を書き換える。存在しなければ何もしない。
func (r *Registry) Update(channelID snowflake.ID, f func(e *Entry)) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.channels[channelID]
	if !ok {
		return false
	}
	f(&e)
	r.channels[channelID] = e
	return true
}

// Delete はチャンネルを監視対象から外す。
func (r *Registry) Delete(channelID snowflake.ID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.channels, channelID)
}

// DeleteGuild はサーバー内の全チャンネルを監視対象から外す。
func (r *Registry) DeleteGuild(guildID snowflake.ID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, e := range r.channels {
		if e.GuildID == guildID {
			delete(r.channels, id)
		}
	}
}

// All は全設定のスナップショットを返す。
func (r *Registry) All() []Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Entry, 0, len(r.channels))
	for _, e := range r.channels {
		out = append(out, e)
	}
	return out
}

// Guild はサーバー内の設定のスナップショットを返す。
func (r *Registry) Guild(guildID snowflake.ID) []Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Entry
	for _, e := range r.channels {
		if e.GuildID == guildID {
			out = append(out, e)
		}
	}
	return out
}
