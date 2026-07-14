// Command irc-bridge bridges a single IRC channel and an AIM conversation. IRC
// channel messages (except the bridge's own) are relayed to the configured AIM
// screen name; AIM instant messages are relayed to the IRC channel as
// PRIVMSGs.
//
// Configuration is via environment variables (see [Config]); set IRC_TLS=true
// to connect to IRC over TLS. The bridge shuts down gracefully on SIGINT /
// SIGTERM.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mk6i/open-oscar-server/client/toc"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("%v", err)
	}
	if err := run(cfg); err != nil {
		log.Fatalf("%v", err)
	}
}

// run wires and runs the bridge until a signal is received or either side
// disconnects. It performs graceful shutdown: it QUITs IRC and closes the TOC
// client before returning.
func run(cfg Config) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	irc, err := DialIRC(ctx, cfg.IRCServer, cfg.IRCPort, cfg.IRCTLS, cfg.IRCNick)
	if err != nil {
		return fmt.Errorf("connect irc: %w", err)
	}
	defer irc.Close()
	if err := irc.Register(cfg.IRCChannel); err != nil {
		return fmt.Errorf("register irc: %w", err)
	}
	log.Printf("irc: connected to %s as %s, joining %s", cfg.IRCServer, cfg.IRCNick, cfg.IRCChannel)

	bridge := &Bridge{
		SelfNick: cfg.IRCNick,
		Channel:  cfg.IRCChannel,
		AIMTo:    cfg.AIMTo,
		IRC:      irc,
	}

	tocClient, err := toc.Dial(cfg.TOCBOTServer, toc.Options{
		Handler:   bridge,
		KeepAlive: 60 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("connect toc: %w", err)
	}
	defer tocClient.Close()
	bridge.AIM = tocClient

	if err := tocClient.SignIn(cfg.ScreenName, cfg.Password); err != nil {
		var signInErr *toc.SignInError
		if errors.As(err, &signInErr) {
			return fmt.Errorf("toc sign in rejected (ERROR:%s) - check credentials", signInErr.Code)
		}
		return fmt.Errorf("toc sign in: %w", err)
	}
	log.Printf("toc: signed in as %s", tocClient.ScreenName())

	tocDone := make(chan error, 1)
	go func() {
		tocDone <- tocClient.Receive(ctx)
	}()

	ircDone := make(chan error, 1)
	go func() {
		ircDone <- ircReadLoop(ctx, irc, bridge)
	}()

	select {
	case <-ctx.Done():
		log.Printf("shutdown signal received")
	case err := <-ircDone:
		log.Printf("irc: %v", err)
	case err := <-tocDone:
		log.Printf("toc: %v", err)
	}

	irc.Quit()
	return nil
}

// ircReadLoop reads IRC lines, replying to PINGs and relaying channel PRIVMSGs
// (from senders other than the bridge) to AIM. It returns when the connection
// is closed or ctx is canceled.
func ircReadLoop(ctx context.Context, irc *IRCConn, bridge *Bridge) error {
	for {
		line, err := irc.ReadLine()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		if line == "" {
			continue
		}
		sender, command, params, trailing := ParseIRCLine(line)
		switch {
		case IsPing(command):
			if err := irc.SendRaw(PongResponse(trailing)); err != nil {
				return err
			}
		case strings.EqualFold(command, "PRIVMSG") && len(params) > 0:
			if err := bridge.HandleIRCPrivmsg(sender, trailing); err != nil {
				log.Printf("aim relay failed: %v", err)
			}
		}
	}
}
