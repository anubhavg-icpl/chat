package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseIRCLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		sender   string
		command  string
		params   []string
		trailing string
	}{
		{
			name:     "PING with token",
			line:     "PING :irc.libera.chat\r\n",
			sender:   "",
			command:  "PING",
			params:   nil,
			trailing: "irc.libera.chat",
		},
		{
			name:     "PRIVMSG with trailing text",
			line:     ":alice!~alice@host.example PRIVMSG #chat :hello world\r\n",
			sender:   "alice",
			command:  "PRIVMSG",
			params:   []string{"#chat"},
			trailing: "hello world",
		},
		{
			name:    "PRIVMSG without trailing",
			line:    ":bob!bob@host.example PRIVMSG #chat\r\n",
			sender:  "bob",
			command: "PRIVMSG",
			params:  []string{"#chat"},
		},
		{
			name:    "JOIN",
			line:    ":carol!carol@host.example JOIN #chat\r\n",
			sender:  "carol",
			command: "JOIN",
			params:  []string{"#chat"},
		},
		{
			name:     "server numeric welcome",
			line:     ":irc.libera.chat 001 bridgebot :Welcome to the Libera.Chat IRC Network\r\n",
			sender:   "irc.libera.chat",
			command:  "001",
			params:   []string{"bridgebot"},
			trailing: "Welcome to the Libera.Chat IRC Network",
		},
		{
			name:   "malformed empty line",
			line:   "",
			sender: "",
		},
		{
			name:   "malformed prefix only",
			line:   ":lonelyprefix",
			sender: "lonelyprefix",
		},
		{
			name:     "trailing text containing colon is preserved",
			line:     ":dave!d@h PRIVMSG #chat :time is 12:30 now\r\n",
			sender:   "dave",
			command:  "PRIVMSG",
			params:   []string{"#chat"},
			trailing: "time is 12:30 now",
		},
		{
			name:    "MODE with multiple params and no trailing",
			line:    ":dave!d@h MODE #chat +o eve\r\n",
			sender:  "dave",
			command: "MODE",
			params:  []string{"#chat", "+o", "eve"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender, command, params, trailing := ParseIRCLine(tt.line)
			assert.Equal(t, tt.sender, sender)
			assert.Equal(t, tt.command, command)
			assert.Equal(t, tt.params, params)
			assert.Equal(t, tt.trailing, trailing)
		})
	}
}

func TestIsPing(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want bool
	}{
		{"uppercase PING", "PING", true},
		{"lowercase ping", "ping", true},
		{"PRIVMSG", "PRIVMSG", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsPing(tt.cmd))
		})
	}
}

func TestPongResponse(t *testing.T) {
	tests := []struct {
		name     string
		trailing string
		want     string
	}{
		{"with token", "irc.libera.chat", "PONG :irc.libera.chat"},
		{"empty token", "", "PONG"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, PongResponse(tt.trailing))
		})
	}
}
