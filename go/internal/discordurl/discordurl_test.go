package discordurl

import "testing"

func TestParseMessageURL(t *testing.T) {
	ref, err := ParseMessageURL("https://discord.com/channels/111/222/333")
	if err != nil {
		t.Fatal(err)
	}
	if ref.GuildID != 111 || ref.ChannelID != 222 || ref.MessageID != 333 {
		t.Fatalf("unexpected: %+v", ref)
	}
	if _, err := ParseMessageURL("https://ptb.discord.com/channels/1/2/3"); err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"", "abc", "http://discord.com/channels/1/2/3", "https://example.com/channels/1/2/3", "https://discord.com/channels/1/2"} {
		if _, err := ParseMessageURL(s); err == nil {
			t.Errorf("expected error: %q", s)
		}
	}
}
