package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
)

// ParseIRCLine decodes a single IRC protocol line into its parts. The returned
// sender is the nickname component of the message prefix (the part before the
// first '!' or '@'); for server-originated numerics it is the server name. line
// may include a trailing CR/LF, which is ignored. Malformed or empty input
// yields zero values without panicking.
func ParseIRCLine(line string) (sender, command string, params []string, trailing string) {
	line = strings.TrimRight(line, "\r\n")

	var prefix string
	if strings.HasPrefix(line, ":") {
		if sp := strings.IndexByte(line, ' '); sp >= 0 {
			prefix = line[1:sp]
			line = line[sp+1:]
		} else {
			prefix = line[1:]
			line = ""
		}
	}
	if prefix != "" {
		if cut := strings.IndexAny(prefix, "!@"); cut >= 0 {
			sender = prefix[:cut]
		} else {
			sender = prefix
		}
	}

	// The trailing parameter begins after the first " :" sequence; everything
	// following it (including further colons) is preserved verbatim.
	if idx := strings.Index(line, " :"); idx >= 0 {
		trailing = line[idx+2:]
		line = line[:idx]
	}

	parts := strings.Fields(line)
	if len(parts) == 0 {
		return sender, "", nil, trailing
	}
	command = parts[0]
	if len(parts) > 1 {
		params = parts[1:]
	}
	return sender, command, params, trailing
}

// IsPing reports whether command is a PING (case-insensitive, per RFC 2812).
func IsPing(command string) bool {
	return strings.EqualFold(command, "PING")
}

// PongResponse builds the reply for a received PING. If the PING carried a
// token it is echoed back; otherwise a bare PONG is returned.
func PongResponse(trailing string) string {
	if trailing == "" {
		return "PONG"
	}
	return "PONG :" + trailing
}

// IRCConn is a minimal IRC client connection over a TCP (optionally TLS)
// socket. Writes are serialized by a mutex so command methods may be called
// concurrently from goroutines other than the read loop.
type IRCConn struct {
	mu     sync.Mutex
	conn   net.Conn
	reader *bufio.Reader
	nick   string
}

// DialIRC connects to server:port (TLS when useTLS is true) and returns a ready
// IRCConn. Registration (NICK/USER) and joining are performed separately via
// [IRCConn.Register].
func DialIRC(ctx context.Context, server string, port int, useTLS bool, nick string) (*IRCConn, error) {
	addr := net.JoinHostPort(server, strconv.Itoa(port))
	if useTLS {
		d := &tls.Dialer{Config: &tls.Config{ServerName: server}}
		conn, err := d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("irc: tls dial %s: %w", addr, err)
		}
		return &IRCConn{conn: conn, reader: bufio.NewReader(conn), nick: nick}, nil
	}
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("irc: dial %s: %w", addr, err)
	}
	return &IRCConn{conn: conn, reader: bufio.NewReader(conn), nick: nick}, nil
}

// Register sends NICK and USER to identify the connection and JOINs channel. It
// does not wait for server acknowledgement.
func (c *IRCConn) Register(channel string) error {
	if err := c.SendRaw("NICK " + c.nick); err != nil {
		return fmt.Errorf("irc: send NICK: %w", err)
	}
	if err := c.SendRaw(fmt.Sprintf("USER %s 0 * :%s", c.nick, c.nick)); err != nil {
		return fmt.Errorf("irc: send USER: %w", err)
	}
	if err := c.SendRaw("JOIN " + channel); err != nil {
		return fmt.Errorf("irc: send JOIN: %w", err)
	}
	return nil
}

// SendRaw writes a single IRC line, appending the required CR/LF terminator.
func (c *IRCConn) SendRaw(line string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := fmt.Fprintf(c.conn, "%s\r\n", line); err != nil {
		return fmt.Errorf("irc: write: %w", err)
	}
	return nil
}

// SendPrivmsg sends a PRIVMSG to target, satisfying the PrivMsgSender interface.
func (c *IRCConn) SendPrivmsg(target, text string) error {
	return c.SendRaw("PRIVMSG " + target + " :" + text)
}

// ReadLine blocks until the next line is available. The returned string keeps
// its trailing newline; callers should strip it via [ParseIRCLine].
func (c *IRCConn) ReadLine() (string, error) {
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("irc: read: %w", err)
	}
	return line, nil
}

// Quit sends a best-effort QUIT message so the server logs a clean departure.
func (c *IRCConn) Quit() {
	_ = c.SendRaw("QUIT :irc-bridge shutting down")
}

// Close closes the underlying connection.
func (c *IRCConn) Close() error {
	return c.conn.Close()
}
