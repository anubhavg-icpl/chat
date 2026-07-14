// Command tocbot is a small, runnable TOC bot built on the client/toc package.
//
// It connects to an Open OSCAR Server TOC endpoint, signs in, and responds to a
// few simple instant-message commands. Configuration is via environment
// variables:
//
//	TOCBOT_SERVER     - TOC server address (default "127.0.0.1:9898")
//	TOCBOT_SCREENNAME - screen name to sign in with (required)
//	TOCBOT_PASSWORD   - password for the screen name (required)
//	TOCBOT_AWAY       - optional away message; empty means online (default online)
//
// The bot reconnects with exponential backoff after a disconnect.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mk6i/open-oscar-server/client/toc"
)

func main() {
	cfg := loadConfig()

	backoff := initialBackoff
	for {
		err := run(cfg)
		if err != nil {
			log.Printf("session ended: %v", err)
		}
		log.Printf("reconnecting in %s", backoff)
		time.Sleep(backoff)
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

// run performs one full connect/sign-in/receive cycle.
func run(cfg config) error {
	handler := &botHandler{away: cfg.Away}
	client, err := toc.Dial(cfg.Server, toc.Options{
		Handler:   handler,
		KeepAlive: 60 * time.Second,
	})
	if err != nil {
		return err
	}
	defer client.Close()
	handler.client = client

	log.Printf("connected to %s", cfg.Server)

	if err := client.SignIn(cfg.ScreenName, cfg.Password); err != nil {
		var signInErr *toc.SignInError
		if errors.As(err, &signInErr) {
			return fmt.Errorf("sign in rejected (ERROR:%s) - check credentials", signInErr.Code)
		}
		return fmt.Errorf("sign in: %w", err)
	}
	log.Printf("signed in as %s", client.ScreenName())

	if cfg.Away == "" {
		if err := client.SetAway(""); err != nil {
			return fmt.Errorf("set online: %w", err)
		}
		log.Printf("status: online")
	} else {
		if err := client.SetAway(cfg.Away); err != nil {
			return fmt.Errorf("set away: %w", err)
		}
		log.Printf("status: away")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return client.Receive(ctx)
}

// botHandler implements toc.Handler. It holds the away message and, once
// connected, a reference to the client used to send replies.
type botHandler struct {
	client *toc.Client
	away   string
}

func (h *botHandler) OnIM(from, text string, autoResponse bool) {
	if autoResponse {
		return
	}
	text = strings.TrimSpace(text)
	log.Printf("IM from %s: %q", from, text)
	if h.client == nil {
		return
	}
	reply := respond(text)
	if err := h.client.SendIM(from, reply); err != nil {
		log.Printf("reply to %s failed: %v", from, err)
	}
}

func (h *botHandler) OnError(code string) {
	log.Printf("server error: ERROR:%s", code)
}

// respond maps an incoming message to a reply using the bot's command set.
func respond(text string) string {
	switch {
	case text == "!help":
		return "commands: !help - show commands, !echo <text> - echo text, " +
			"!time - current time, !ping - reply pong"
	case text == "!ping":
		return "pong"
	case text == "!time":
		return "current time is " + time.Now().Format(time.RFC1123)
	case strings.HasPrefix(text, "!echo "):
		return strings.TrimPrefix(text, "!echo ")
	case text == "!echo":
		return ""
	default:
		return text
	}
}

const (
	initialBackoff = 2 * time.Second
	maxBackoff     = 60 * time.Second
)

// config holds resolved environment configuration.
type config struct {
	Server     string
	ScreenName string
	Password   string
	Away       string
}

func loadConfig() config {
	c := config{
		Server:     envOrDefault("TOCBOT_SERVER", "127.0.0.1:9898"),
		ScreenName: os.Getenv("TOCBOT_SCREENNAME"),
		Password:   os.Getenv("TOCBOT_PASSWORD"),
		Away:       os.Getenv("TOCBOT_AWAY"),
	}
	if c.ScreenName == "" || c.Password == "" {
		log.Fatal("TOCBOT_SCREENNAME and TOCBOT_PASSWORD must be set")
	}
	return c
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
