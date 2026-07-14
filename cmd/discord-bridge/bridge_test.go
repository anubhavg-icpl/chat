package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// fakeAIMSender records SendIM calls for assertion.
type fakeAIMSender struct {
	sends []aimSend
	err   error
}

type aimSend struct {
	to   string
	text string
}

func (f *fakeAIMSender) SendIM(to, text string) error {
	f.sends = append(f.sends, aimSend{to: to, text: text})
	return f.err
}

// fakeDiscordSender records Send calls for assertion.
type fakeDiscordSender struct {
	sends []discordSend
	err   error
}

type discordSend struct {
	channelID string
	content   string
}

func (f *fakeDiscordSender) Send(channelID, content string) error {
	f.sends = append(f.sends, discordSend{channelID: channelID, content: content})
	return f.err
}

func TestFormatDiscordToAIM(t *testing.T) {
	tests := []struct {
		name    string
		author  string
		content string
		want    string
	}{
		{
			name:    "simple message",
			author:  "alice",
			content: "hello there",
			want:    "alice: hello there",
		},
		{
			name:    "empty content",
			author:  "bob",
			content: "",
			want:    "bob: ",
		},
		{
			name:    "unicode content",
			author:  "carol",
			content: "héllo 世界 🎉",
			want:    "carol: héllo 世界 🎉",
		},
		{
			name:    "truncated when too long",
			author:  "dave",
			content: string(repeatRune('x', maxAIMRunes)),
			want:    "dave: " + string(repeatRune('x', maxAIMRunes-len("dave: ")-1)) + "…",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, formatDiscordToAIM(tc.author, tc.content))
		})
	}
}

func TestFormatAIMToDiscord(t *testing.T) {
	tests := []struct {
		name string
		from string
		text string
		want string
	}{
		{
			name: "simple message",
			from: "eve",
			text: "hi from aim",
			want: "eve: hi from aim",
		},
		{
			name: "truncated when too long",
			from: "frank",
			text: string(repeatRune('y', maxDiscordRunes)),
			want: "frank: " + string(repeatRune('y', maxDiscordRunes-len("frank: ")-1)) + "…",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, formatAIMToDiscord(tc.from, tc.text))
		})
	}
}

func TestShouldRelayDiscord(t *testing.T) {
	const watched = "chan-1"
	const bot = "bot-42"
	tests := []struct {
		name        string
		channel     string
		author      string
		trigger     string
		content     string
		wantRelayed bool
	}{
		{
			name:        "eligible message relayed",
			channel:     watched,
			author:      "user-1",
			content:     "hello",
			wantRelayed: true,
		},
		{
			name:        "wrong channel ignored",
			channel:     "other-channel",
			author:      "user-1",
			content:     "hello",
			wantRelayed: false,
		},
		{
			name:        "own bot message ignored",
			channel:     watched,
			author:      bot,
			content:     "hello",
			wantRelayed: false,
		},
		{
			name:        "empty channel ignored",
			channel:     "",
			author:      "user-1",
			content:     "hello",
			wantRelayed: false,
		},
		{
			name:        "trigger matched relayed",
			channel:     watched,
			author:      "user-1",
			trigger:     "!aim ",
			content:     "!aim hello",
			wantRelayed: true,
		},
		{
			name:        "trigger not matched ignored",
			channel:     watched,
			author:      "user-1",
			trigger:     "!aim ",
			content:     "hello",
			wantRelayed: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldRelayDiscord(watched, bot, tc.trigger, tc.channel, tc.author, tc.content)
			assert.Equal(t, tc.wantRelayed, got)
		})
	}
}

func TestBridge_HandleDiscordMessage_RoutesToAIM(t *testing.T) {
	aim := &fakeAIMSender{}
	b := &Bridge{
		Discord:   &fakeDiscordSender{},
		AIM:       aim,
		ChannelID: "chan-1",
		AIMTarget: "aim-buddy",
		BotID:     "bot-42",
	}

	b.HandleDiscordMessage("chan-1", "user-1", "alice", "hello world")

	assert.Len(t, aim.sends, 1)
	assert.Equal(t, aimSend{to: "aim-buddy", text: "alice: hello world"}, aim.sends[0])
}

func TestBridge_HandleDiscordMessage_TriggerStripsPrefix(t *testing.T) {
	aim := &fakeAIMSender{}
	b := &Bridge{
		Discord:   &fakeDiscordSender{},
		AIM:       aim,
		ChannelID: "chan-1",
		AIMTarget: "aim-buddy",
		BotID:     "bot-42",
		Trigger:   "!aim ",
	}

	b.HandleDiscordMessage("chan-1", "user-1", "alice", "!aim hello")

	assert.Equal(t, []aimSend{{to: "aim-buddy", text: "alice: hello"}}, aim.sends)
}

func TestBridge_HandleDiscordMessage_SkipsBotAndWrongChannel(t *testing.T) {
	aim := &fakeAIMSender{}
	b := &Bridge{
		Discord:   &fakeDiscordSender{},
		AIM:       aim,
		ChannelID: "chan-1",
		AIMTarget: "aim-buddy",
		BotID:     "bot-42",
	}

	b.HandleDiscordMessage("chan-1", "bot-42", "bridge", "echo") // own bot
	b.HandleDiscordMessage("chan-2", "user-1", "alice", "nope")  // wrong channel

	assert.Empty(t, aim.sends)
}

func TestBridge_HandleAIMMessage_RoutesToDiscord(t *testing.T) {
	discord := &fakeDiscordSender{}
	b := &Bridge{
		Discord:   discord,
		AIM:       &fakeAIMSender{},
		ChannelID: "chan-1",
		AIMTarget: "aim-buddy",
		BotID:     "bot-42",
	}

	b.HandleAIMMessage("aim-buddy", "hi from aim")

	assert.Equal(t, []discordSend{{channelID: "chan-1", content: "aim-buddy: hi from aim"}}, discord.sends)
}

func TestTruncateRunes(t *testing.T) {
	tests := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{name: "under limit unchanged", in: "abc", max: 10, want: "abc"},
		{name: "at limit unchanged", in: "abc", max: 3, want: "abc"},
		{name: "over limit truncated with ellipsis", in: "abcdef", max: 4, want: "abc…"},
		{name: "rune aware", in: "世界世界", max: 3, want: "世界…"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, truncateRunes(tc.in, tc.max))
		})
	}
}

func repeatRune(r rune, n int) []rune {
	out := make([]rune, n)
	for i := range out {
		out[i] = r
	}
	return out
}
