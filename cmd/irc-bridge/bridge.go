package main

import "log"

// IMSender sends an instant message to a recipient. The TOC client
// (*toc.Client) satisfies this interface.
type IMSender interface {
	SendIM(to, text string) error
}

// PrivMsgSender sends a PRIVMSG to an IRC target. The IRCConn type satisfies
// this interface.
type PrivMsgSender interface {
	SendPrivmsg(target, text string) error
}

// formatIRCtoAIM renders an IRC channel message as the body of an AIM instant
// message, prefixed with the originating IRC nick.
func formatIRCtoAIM(nick, text string) string {
	return nick + ": " + text
}

// formatAIMtoIRC renders a received AIM instant message as the body of an IRC
// PRIVMSG, prefixed with the originating AIM screen name.
func formatAIMtoIRC(from, text string) string {
	return from + ": " + text
}

// Bridge relays traffic between an IRC channel and an AIM conversation. Its
// routing methods are pure (they neither log nor touch the network directly)
// and depend only on the small [IMSender] and [PrivMsgSender] interfaces, so
// they can be unit tested with stubs.
type Bridge struct {
	// SelfNick is the bridge's own IRC nick; messages from it are not relayed
	// to AIM to avoid loops.
	SelfNick string
	// Channel is the IRC channel PRIVMSGs are posted to for AIM-originated
	// messages.
	Channel string
	// AIMTo is the AIM screen name that receives IRC channel traffic.
	AIMTo string
	// AIM sends instant messages (the IRC -> AIM direction).
	AIM IMSender
	// IRC sends PRIVMSGs (the AIM -> IRC direction).
	IRC PrivMsgSender
}

// HandleIRCPrivmsg relays an IRC channel PRIVMSG to the configured AIM
// recipient, unless the sender is the bridge itself.
func (b *Bridge) HandleIRCPrivmsg(sender, text string) error {
	if sender == b.SelfNick {
		return nil
	}
	return b.AIM.SendIM(b.AIMTo, formatIRCtoAIM(sender, text))
}

// HandleAIMIM relays a received AIM instant message to the IRC channel.
func (b *Bridge) HandleAIMIM(from, text string) error {
	return b.IRC.SendPrivmsg(b.Channel, formatAIMtoIRC(from, text))
}

// OnIM implements toc.Handler by relaying a received AIM instant message to the
// IRC channel. The auto flag (away replies) is ignored: all IMs are relayed.
func (b *Bridge) OnIM(from, text string, auto bool) {
	if err := b.HandleAIMIM(from, text); err != nil {
		log.Printf("irc relay failed: %v", err)
	}
}

// OnError implements toc.Handler by logging TOC server errors.
func (b *Bridge) OnError(code string) {
	log.Printf("toc server error: ERROR:%s", code)
}
