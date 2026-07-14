// Command discord-bridge relays messages between a Discord channel and an AIM
// user via the TOC protocol.
//
// Messages sent in the watched Discord channel are delivered to the configured
// AIM screen name as an instant message, and instant messages received by the
// bridge's AIM account are posted into the watched Discord channel.
//
// Configuration is via environment variables:
//
//	DISCORD_TOKEN        - Discord bot token (required)
//	DISCORD_CHANNEL_ID   - channel to bridge (required)
//	AIM_TO               - AIM screen name receiving Discord->AIM messages (required)
//	TOCBOT_SERVER        - TOC server address (default "127.0.0.1:9898")
//	TOCBOT_SCREENNAME    - the bridge's AIM account (required)
//	TOCBOT_PASSWORD      - password for the AIM account (required)
//	BRIDGE_TRIGGER       - optional prefix a Discord message must start with to
//	                       be relayed; empty relays all non-bot messages
package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/mk6i/open-oscar-server/client/toc"
)

func main() {
	cfg := loadConfig()

	bridge := &Bridge{
		ChannelID: cfg.DiscordChannelID,
		AIMTarget: cfg.AIMTo,
		Trigger:   cfg.Trigger,
	}

	dg, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		log.Fatalf("discord setup: %v", err)
	}
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentMessageContent
	dg.AddHandler(discordMessageHandler(bridge))
	bridge.Discord = discordAdapter{session: dg}

	client, err := toc.Dial(cfg.TOCServer, toc.Options{
		Handler:   &aimHandler{bridge: bridge},
		KeepAlive: 60 * time.Second,
	})
	if err != nil {
		log.Fatalf("toc dial: %v", err)
	}
	bridge.AIM = client

	if err := client.SignIn(cfg.TOCScreenName, cfg.TOCPassword); err != nil {
		var signInErr *toc.SignInError
		if errors.As(err, &signInErr) {
			log.Fatalf("aim sign in rejected (ERROR:%s) - check credentials", signInErr.Code)
		}
		log.Fatalf("aim sign in: %v", err)
	}
	log.Printf("signed in to AIM as %s", client.ScreenName())

	if err := dg.Open(); err != nil {
		log.Fatalf("discord open: %v", err)
	}
	if dg.State.User != nil {
		bridge.BotID = dg.State.User.ID
	}
	log.Printf("connected to Discord; bridging channel %s <-> AIM %s", cfg.DiscordChannelID, cfg.AIMTo)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	tocErr := make(chan error, 1)
	go func() {
		tocErr <- client.Receive(ctx)
	}()

	select {
	case <-ctx.Done():
		if ctx.Err() != nil {
			log.Printf("shutdown signal received: %v", ctx.Err())
		}
	case err := <-tocErr:
		if err != nil {
			log.Printf("toc session ended: %v", err)
		}
	}

	if err := client.Close(); err != nil {
		log.Printf("toc close: %v", err)
	}
	if err := dg.Close(); err != nil {
		log.Printf("discord close: %v", err)
	}
}

// discordMessageHandler returns a Discord message-create handler that forwards
// eligible messages to the bridge.
func discordMessageHandler(bridge *Bridge) func(*discordgo.Session, *discordgo.MessageCreate) {
	return func(_ *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author == nil {
			return
		}
		bridge.HandleDiscordMessage(m.ChannelID, m.Author.ID, m.Author.Username, m.Content)
	}
}

// aimHandler implements toc.Handler, forwarding incoming AIM messages to the
// bridge.
type aimHandler struct {
	bridge *Bridge
}

// OnIM relays a non-automatic AIM instant message into Discord.
func (h *aimHandler) OnIM(from, text string, auto bool) {
	if auto {
		return
	}
	h.bridge.HandleAIMMessage(from, text)
}

// OnError logs a TOC server error code.
func (h *aimHandler) OnError(code string) {
	log.Printf("aim server error: ERROR:%s", code)
}

// config holds resolved environment configuration for the bridge.
type config struct {
	DiscordToken     string
	DiscordChannelID string
	AIMTo            string
	TOCServer        string
	TOCScreenName    string
	TOCPassword      string
	Trigger          string
}

func loadConfig() config {
	c := config{
		DiscordToken:     os.Getenv("DISCORD_TOKEN"),
		DiscordChannelID: os.Getenv("DISCORD_CHANNEL_ID"),
		AIMTo:            os.Getenv("AIM_TO"),
		TOCServer:        envOrDefault("TOCBOT_SERVER", "127.0.0.1:9898"),
		TOCScreenName:    os.Getenv("TOCBOT_SCREENNAME"),
		TOCPassword:      os.Getenv("TOCBOT_PASSWORD"),
		Trigger:          os.Getenv("BRIDGE_TRIGGER"),
	}
	var missing []string
	if c.DiscordToken == "" {
		missing = append(missing, "DISCORD_TOKEN")
	}
	if c.DiscordChannelID == "" {
		missing = append(missing, "DISCORD_CHANNEL_ID")
	}
	if c.AIMTo == "" {
		missing = append(missing, "AIM_TO")
	}
	if c.TOCScreenName == "" {
		missing = append(missing, "TOCBOT_SCREENNAME")
	}
	if c.TOCPassword == "" {
		missing = append(missing, "TOCBOT_PASSWORD")
	}
	if len(missing) > 0 {
		log.Fatalf("missing required environment variables: %v", missing)
	}
	return c
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
