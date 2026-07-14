package main

import (
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// maxAIMRunes caps a relayed message's length for the AIM direction. AIM
// clients and TOC gateways have practical size limits; truncating avoids
// dropping overly long Discord messages.
const maxAIMRunes = 2000

// maxDiscordRunes caps a relayed message's length for the Discord direction.
// Discord rejects messages over 2000 characters, so truncating keeps the post
// from failing.
const maxDiscordRunes = 2000

// AIMSender sends an instant message to an AIM user over TOC. It is satisfied
// by *toc.Client.
type AIMSender interface {
	SendIM(to, text string) error
}

// DiscordSender posts a message into a Discord channel. It is satisfied by the
// discordAdapter that wraps a *discordgo.Session.
type DiscordSender interface {
	Send(channelID, content string) error
}

// Bridge relays messages between a watched Discord channel and a single AIM
// user. The Discord and AIM dependencies are small interfaces so routing can be
// unit tested without live network connections.
type Bridge struct {
	Discord   DiscordSender
	AIM       AIMSender
	ChannelID string
	AIMTarget string
	BotID     string
	// Trigger is an optional prefix a Discord message must start with to be
	// relayed to AIM. When empty, every eligible message is relayed.
	Trigger string
}

// HandleDiscordMessage relays a Discord message to AIM when it passes the
// routing rules (correct channel, not authored by the bot, matches the optional
// trigger). The trigger prefix, if any, is stripped before sending.
func (b *Bridge) HandleDiscordMessage(channelID, authorID, authorName, content string) {
	if !shouldRelayDiscord(b.ChannelID, b.BotID, b.Trigger, channelID, authorID, content) {
		return
	}
	body := content
	if b.Trigger != "" {
		body = strings.TrimPrefix(content, b.Trigger)
	}
	if err := b.AIM.SendIM(b.AIMTarget, formatDiscordToAIM(authorName, body)); err != nil {
		log.Printf("discord->aim send failed: %v", err)
	}
}

// HandleAIMMessage posts an AIM message into the bridged Discord channel.
func (b *Bridge) HandleAIMMessage(from, text string) {
	if err := b.Discord.Send(b.ChannelID, formatAIMToDiscord(from, text)); err != nil {
		log.Printf("aim->discord send failed: %v", err)
	}
}

// shouldRelayDiscord reports whether a Discord message should be bridged to AIM.
// A message is relayed when it is in the watched channel, was not authored by
// the bridge bot, and either there is no trigger or the content starts with it.
func shouldRelayDiscord(watchedChannelID, botID, trigger, msgChannelID, authorID, content string) bool {
	if msgChannelID != watchedChannelID {
		return false
	}
	if authorID == botID {
		return false
	}
	if trigger != "" && !strings.HasPrefix(content, trigger) {
		return false
	}
	return true
}

// formatDiscordToAIM formats a Discord message for delivery over AIM as
// "<author>: <content>", truncating to maxAIMRunes runes.
func formatDiscordToAIM(author, content string) string {
	return truncateRunes(author+": "+content, maxAIMRunes)
}

// formatAIMToDiscord formats an AIM message for posting to Discord as
// "<from>: <text>", truncating to maxDiscordRunes runes.
func formatAIMToDiscord(from, text string) string {
	return truncateRunes(from+": "+text, maxDiscordRunes)
}

// truncateRunes returns s shortened to at most max runes, appending an ellipsis
// when truncation occurs. max must be greater than 1.
func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

// discordAdapter adapts a *discordgo.Session to the DiscordSender interface.
type discordAdapter struct {
	session *discordgo.Session
}

// Send posts content to the given channel via the underlying Discord session.
func (d discordAdapter) Send(channelID, content string) error {
	_, err := d.session.ChannelMessageSend(channelID, content)
	return err
}
