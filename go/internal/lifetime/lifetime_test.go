package lifetime

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	day := 24 * time.Hour
	tests := []struct {
		in   string
		want time.Duration
	}{
		{"1week", 7 * day},
		{"1w", 7 * day},
		{"1day", day},
		{"1d", day},
		{"1hour", time.Hour},
		{"1h", time.Hour},
		{"1min", time.Minute},
		{"1mi", time.Minute},
		{"1minute", time.Minute},
		{"1sec", time.Second},
		{"1s", time.Second},
		{"2days", 2 * day},
		{"1week 2day 3hour 4min 5sec", 9*day + 3*time.Hour + 4*time.Minute + 5*time.Second},
		{"1 day 1 hour 30 min", day + time.Hour + 30*time.Minute},
		{"90mi", 90 * time.Minute},
		{"1h90mi", 2*time.Hour + 30*time.Minute},
		{"1DAY", day},
	}
	for _, tt := range tests {
		got, err := Parse(tt.in)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("Parse(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestParseError(t *testing.T) {
	for _, in := range []string{"", "   ", "1day 2day", "abc", "1day foo", "0s", "99999999999999999999d", "5000week"} {
		if _, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) expected error", in)
		}
	}
}

func TestFormat(t *testing.T) {
	d := 9*24*time.Hour + 3*time.Hour + 4*time.Minute + 5*time.Second
	if got := Format(d, "ja"); got != "9日3時間4分5秒" {
		t.Errorf("ja: %q", got)
	}
	if got := Format(d, "en"); got != "9d 3h 4m 5s" {
		t.Errorf("en: %q", got)
	}
	if got := Format(time.Hour, "ja"); got != "1時間" {
		t.Errorf("ja: %q", got)
	}
}
