package main

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

// Config holds resolved environment configuration for the bridge. It is loaded
// via envconfig with an empty prefix, so field tags name the env vars directly.
type Config struct {
	IRCServer    string `envconfig:"IRC_SERVER" required:"true"`
	IRCPort      int    `envconfig:"IRC_PORT" default:"6667"`
	IRCTLS       bool   `envconfig:"IRC_TLS" default:"false"`
	IRCNick      string `envconfig:"IRC_NICK" required:"true"`
	IRCChannel   string `envconfig:"IRC_CHANNEL" required:"true"`
	AIMTo        string `envconfig:"AIM_TO" required:"true"`
	TOCBOTServer string `envconfig:"TOCBOT_SERVER" default:"127.0.0.1:9898"`
	ScreenName   string `envconfig:"TOCBOT_SCREENNAME" required:"true"`
	Password     string `envconfig:"TOCBOT_PASSWORD" required:"true"`
}

// loadConfig reads configuration from environment variables.
func loadConfig() (Config, error) {
	var c Config
	if err := envconfig.Process("", &c); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	return c, nil
}
