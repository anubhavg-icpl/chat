package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mk6i/open-oscar-server/client/toc"
)

// Config holds all aibot settings, populated from environment variables.
type Config struct {
	BaseURL      string
	APIKey       string
	Model        string
	SystemPrompt string
	TOCServer    string
	ScreenName   string
	Password     string
	HistoryLimit int
}

func main() {
	logger := log.New(os.Stderr, "aibot: ", log.LstdFlags)

	cfg, err := loadConfig()
	if err != nil {
		logger.Fatalf("configuration error: %v", err)
	}

	completer := NewChatClient(cfg.BaseURL, cfg.APIKey, cfg.Model)
	conversation := NewConversation(cfg.HistoryLimit)
	bot := &Bot{
		conversation: conversation,
		completer:    completer,
		systemPrompt: cfg.SystemPrompt,
		timeout:      60 * time.Second,
		log:          logger,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Printf("connecting to TOC server at %s", cfg.TOCServer)
	client, err := toc.Dial(cfg.TOCServer, toc.Options{
		Handler:     bot,
		KeepAlive:   30 * time.Second,
		DialTimeout: 10 * time.Second,
	})
	if err != nil {
		logger.Fatalf("dial: %v", err)
	}

	if err := client.SignIn(cfg.ScreenName, cfg.Password); err != nil {
		logger.Fatalf("sign in: %v", err)
	}
	logger.Printf("signed in as %s; model=%s", cfg.ScreenName, cfg.Model)

	// Wire the TOC client as the reply sender before any message is processed.
	// OnIM is only ever invoked from Receive (started below), so this is set in
	// time and is non-nil when used.
	bot.sender = client

	recvErr := make(chan error, 1)
	go func() {
		recvErr <- client.Receive(ctx)
	}()

	select {
	case <-ctx.Done():
		logger.Printf("shutdown signal received")
	case err := <-recvErr:
		if err != nil {
			logger.Printf("receive loop ended: %v", err)
		}
	}

	if err := client.Close(); err != nil {
		logger.Printf("close: %v", err)
	}
	logger.Printf("bye")
}

// loadConfig reads settings from environment variables, applying defaults and
// validating that the required values are present.
func loadConfig() (Config, error) {
	cfg := Config{
		BaseURL:      getenv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		APIKey:       os.Getenv("OPENAI_API_KEY"),
		Model:        getenv("OPENAI_MODEL", "gpt-4o-mini"),
		SystemPrompt: getenv("AIBOT_SYSTEM_PROMPT", defaultSystemPrompt),
		TOCServer:    getenv("TOCBOT_SERVER", "127.0.0.1:9898"),
		ScreenName:   os.Getenv("TOCBOT_SCREENNAME"),
		Password:     os.Getenv("TOCBOT_PASSWORD"),
		HistoryLimit: getenvInt("AIBOT_HISTORY_LIMIT", 8),
	}

	var missing []string
	for _, c := range []struct{ name, val string }{
		{"TOCBOT_SCREENNAME", cfg.ScreenName},
		{"TOCBOT_PASSWORD", cfg.Password},
		{"OPENAI_API_KEY", cfg.APIKey},
	} {
		if c.val == "" {
			missing = append(missing, c.name)
		}
	}
	if len(missing) > 0 {
		return cfg, fmt.Errorf("required environment variable(s) not set: %s", strings.Join(missing, ", "))
	}
	return cfg, nil
}

// getenv returns the value of the environment variable named by key, or def if
// it is empty or unset.
func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// getenvInt parses the environment variable named by key as an int, returning
// def if it is empty or not a valid integer.
func getenvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
