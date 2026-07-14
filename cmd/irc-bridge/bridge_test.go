package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// stubAIM records every SendIM call, satisfying [IMSender].
type stubAIM struct {
	sent []stubIM
}

type stubIM struct {
	to   string
	text string
}

func (s *stubAIM) SendIM(to, text string) error {
	s.sent = append(s.sent, stubIM{to: to, text: text})
	return nil
}

// stubIRC records every SendPrivmsg call, satisfying [PrivMsgSender].
type stubIRC struct {
	sent []stubPrivmsg
}

type stubPrivmsg struct {
	target string
	text   string
}

func (s *stubIRC) SendPrivmsg(target, text string) error {
	s.sent = append(s.sent, stubPrivmsg{target: target, text: text})
	return nil
}

func TestFormatIRCtoAIM(t *testing.T) {
	tests := []struct {
		name string
		nick string
		text string
		want string
	}{
		{"basic", "alice", "hello there", "alice: hello there"},
		{"empty text", "bob", "", "bob: "},
		{"text with colon", "carol", "see 1:1", "carol: see 1:1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatIRCtoAIM(tt.nick, tt.text))
		})
	}
}

func TestFormatAIMtoIRC(t *testing.T) {
	tests := []struct {
		name string
		from string
		text string
		want string
	}{
		{"basic", "buddy", "hi all", "buddy: hi all"},
		{"empty text", "buddy", "", "buddy: "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatAIMtoIRC(tt.from, tt.text))
		})
	}
}

func TestBridge_HandleIRCPrivmsg_Relays(t *testing.T) {
	aim := &stubAIM{}
	b := &Bridge{SelfNick: "bridgebot", Channel: "#chat", AIMTo: "aimuser", AIM: aim}

	assert.NoError(t, b.HandleIRCPrivmsg("alice", "hello"))

	want := []stubIM{{to: "aimuser", text: "alice: hello"}}
	assert.Equal(t, want, aim.sent)
}

func TestBridge_HandleIRCPrivmsg_SkipsSelf(t *testing.T) {
	aim := &stubAIM{}
	b := &Bridge{SelfNick: "bridgebot", Channel: "#chat", AIMTo: "aimuser", AIM: aim}

	assert.NoError(t, b.HandleIRCPrivmsg("bridgebot", "should not relay"))

	assert.Empty(t, aim.sent)
}

func TestBridge_HandleAIMIM_Relays(t *testing.T) {
	irc := &stubIRC{}
	b := &Bridge{SelfNick: "bridgebot", Channel: "#chat", AIMTo: "aimuser", IRC: irc}

	assert.NoError(t, b.HandleAIMIM("buddy", "hi from aim"))

	want := []stubPrivmsg{{target: "#chat", text: "buddy: hi from aim"}}
	assert.Equal(t, want, irc.sent)
}

func TestBridge_OnIM_RoutesToIRC(t *testing.T) {
	irc := &stubIRC{}
	b := &Bridge{SelfNick: "bridgebot", Channel: "#chat", AIMTo: "aimuser", IRC: irc}

	b.OnIM("buddy", "hello channel", false)

	want := []stubPrivmsg{{target: "#chat", text: "buddy: hello channel"}}
	assert.Equal(t, want, irc.sent)
}
