// Package lifetime はメッセージを残す期間（ライフタイム）文字列の解析と表示を行う。
//
// Python 版の LifeTimeUtil と互換の書式を受け付ける。
//
//	"1week 2day 3hour 4min 5sec", "1 day 1 hour 30 min", "90mi", "1h90mi"
package lifetime

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Max はライフタイムの上限。DB や時刻計算のオーバーフローを避けるための安全弁。
const Max = 10 * 365 * 24 * time.Hour

type unit struct {
	key      string
	duration time.Duration
	ja       string
	en       string
}

var units = []unit{
	{"weeks", 7 * 24 * time.Hour, "週間", "w"},
	{"days", 24 * time.Hour, "日", "d"},
	{"hours", time.Hour, "時間", "h"},
	{"minutes", time.Minute, "分", "m"},
	{"seconds", time.Second, "秒", "s"},
}

// 長い表記から先にマッチさせる（"min" より先に "minute" を試す）。
var aliases = map[string]string{
	"week": "weeks", "w": "weeks",
	"day": "days", "d": "days",
	"hour": "hours", "h": "hours",
	"minute": "minutes", "min": "minutes", "mi": "minutes",
	"second": "seconds", "sec": "seconds", "s": "seconds",
}

var pattern = regexp.MustCompile(`(?i)(\d+)\s*(weeks?|w|days?|d|hours?|h|minutes?|mins?|mi|seconds?|secs?|s)`)

// Parse はライフタイム文字列を time.Duration に変換する。
func Parse(s string) (time.Duration, error) {
	if strings.TrimSpace(s) == "" {
		return 0, errors.New("文字を入力してください。")
	}

	matches := pattern.FindAllStringSubmatchIndex(s, -1)
	counts := map[string]int{}
	var total time.Duration
	rest := []byte(s)

	for _, m := range matches {
		key := normalize(s[m[4]:m[5]])
		counts[key]++
		for _, u := range units {
			if u.key != key {
				continue
			}
			q, err := strconv.ParseInt(s[m[2]:m[3]], 10, 64)
			if err != nil || q > int64(Max/u.duration) {
				return 0, fmt.Errorf("ライフタイムが長すぎます。（最大 %s）", Format(Max, "ja"))
			}
			total += time.Duration(q) * u.duration
		}
		for i := m[0]; i < m[1]; i++ {
			rest[i] = ' '
		}
		if total > Max {
			return 0, fmt.Errorf("ライフタイムが長すぎます。（最大 %s）", Format(Max, "ja"))
		}
	}

	var dup []string
	for _, u := range units {
		if counts[u.key] >= 2 {
			dup = append(dup, u.key)
		}
	}
	if len(dup) > 0 {
		return 0, fmt.Errorf("重複している項目があります。:'%s'", strings.Join(dup, "', '"))
	}

	if invalid := strings.Fields(string(rest)); len(invalid) > 0 {
		return 0, fmt.Errorf("不正な文字が検出されました。:'%s'", strings.Join(invalid, "', '"))
	}

	if total <= 0 {
		return 0, errors.New("ライフタイムが指定されていません。")
	}
	return total, nil
}

func normalize(u string) string {
	u = strings.ToLower(u)
	if k, ok := aliases[u]; ok {
		return k
	}
	// 複数形 (days, mins, secs ...)
	if k, ok := aliases[strings.TrimSuffix(u, "s")]; ok {
		return k
	}
	return u
}

// Format はライフタイムを人が読める文字列にする。
func Format(d time.Duration, lang string) string {
	if d <= 0 {
		if lang == "ja" {
			return "0秒"
		}
		return "0s"
	}
	var parts []string
	for _, u := range units {
		if u.key == "weeks" {
			// 週は日にまとめて表示する
			continue
		}
		n := d / u.duration
		if n == 0 {
			continue
		}
		d -= n * u.duration
		if lang == "ja" {
			parts = append(parts, fmt.Sprintf("%d%s", n, u.ja))
		} else {
			parts = append(parts, fmt.Sprintf("%d%s", n, u.en))
		}
	}
	if lang == "ja" {
		return strings.Join(parts, "")
	}
	return strings.Join(parts, " ")
}
