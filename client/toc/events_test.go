package toc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestParseEvent exercises the message formats produced by server/toc
// (see cmd_server.go convertICBMInstantMsg).
func TestParseEvent(t *testing.T) {
	tests := []struct {
		name string
		msg  string
		want Event
	}{
		{
			name: "IM_IN TOC1",
			msg:  "IM_IN:alice:F:hello world",
			want: Event{Type: EventIM, Raw: "IM_IN:alice:F:hello world", From: "alice", Text: "hello world", Auto: false},
		},
		{
			name: "IM_IN auto response",
			msg:  "IM_IN:bob:T:away!",
			want: Event{Type: EventIM, Raw: "IM_IN:bob:T:away!", From: "bob", Text: "away!", Auto: true},
		},
		{
			name: "IM_IN keeps colons in message",
			msg:  "IM_IN:alice:F:time is 12:30:45",
			want: Event{Type: EventIM, Raw: "IM_IN:alice:F:time is 12:30:45", From: "alice", Text: "time is 12:30:45", Auto: false},
		},
		{
			name: "IM_IN2 TOC2",
			msg:  "IM_IN2:carol:F:F:yo",
			want: Event{Type: EventIM, Raw: "IM_IN2:carol:F:F:yo", From: "carol", Text: "yo", Auto: false},
		},
		{
			name: "IM_IN_ENC2 enhanced",
			msg:  "IM_IN_ENC2:dave:F:F:T:A:F:L:en:encoded message",
			want: Event{Type: EventIM, Raw: "IM_IN_ENC2:dave:F:F:T:A:F:L:en:encoded message", From: "dave", Text: "encoded message", Auto: false},
		},
		{
			name: "ERROR with code only",
			msg:  "ERROR:980",
			want: Event{Type: EventError, Raw: "ERROR:980", Code: "980"},
		},
		{
			name: "ERROR with arguments",
			msg:  "ERROR:901:someuser",
			want: Event{Type: EventError, Raw: "ERROR:901:someuser", Code: "901:someuser"},
		},
		{
			name: "SIGN_ON",
			msg:  "SIGN_ON:TOC1.0",
			want: Event{Type: EventSignOn, Raw: "SIGN_ON:TOC1.0", Name: "TOC1.0"},
		},
		{
			name: "NICK",
			msg:  "NICK:TestUser",
			want: Event{Type: EventNick, Raw: "NICK:TestUser", Name: "TestUser"},
		},
		{
			name: "unknown command is EventOther",
			msg:  "PAUSE:",
			want: Event{Type: EventOther, Raw: "PAUSE:", Name: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseEvent(tt.msg)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSplitColon(t *testing.T) {
	assert.Equal(t, []string{"a", "b", "c"}, splitColon("a:b:c", 3))
	assert.Equal(t, []string{"a", "b:c:d"}, splitColon("a:b:c:d", 2))
	assert.Equal(t, []string{"a", "", ""}, splitColon("a", 3))
}
