package toc

import (
	"bufio"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

// Options configures a [Client]. The zero value is usable but receives no
// events; set Handler or OnEvent to react to incoming messages.
type Options struct {
	// Handler receives typed callbacks for the most common message types. May
	// be nil.
	Handler Handler
	// OnEvent receives every parsed message and is invoked before Handler. May
	// be nil.
	OnEvent EventHandler
	// KeepAlive is the interval at which keep-alive FLAP frames are sent while
	// [Client.Receive] is running. Zero (the default) disables keep-alives.
	KeepAlive time.Duration
	// DialTimeout is the timeout used when establishing the TCP connection in
	// [Dial]. Zero means no timeout.
	DialTimeout time.Duration
}

// SignInError reports a failed sign-on. Code is the text the server sent after
// "ERROR:" (for example "980" for an incorrect screen name or password).
type SignInError struct {
	Code string
}

// Error implements the error interface.
func (e *SignInError) Error() string {
	return fmt.Sprintf("toc: sign in failed (ERROR:%s)", e.Code)
}

// Client is a connected TOC protocol client. A Client is safe for concurrent
// use: command methods (SendIM, SetAway, SendCommand) may be called from any
// goroutine, including from within a [Handler] invoked by [Client.Receive].
type Client struct {
	conn      net.Conn
	reader    *bufio.Reader
	handler   Handler
	onEvent   EventHandler
	keepAlive time.Duration

	mu        sync.Mutex
	connMu    sync.Mutex
	seq       uint16
	screenNm  string
	online    bool
	closed    bool
	closeErr  error
	closeOnce sync.Once
}

// New wraps an existing connection as a TOC [Client] using opts. It is intended
// for advanced uses (for example TLS or in-memory transports). Most callers
// should use [Dial].
func New(conn net.Conn, opts Options) *Client {
	return &Client{
		conn:      conn,
		reader:    bufio.NewReader(conn),
		handler:   opts.Handler,
		onEvent:   opts.OnEvent,
		keepAlive: opts.KeepAlive,
	}
}

// Dial connects to the TOC server at addr (host:port) and returns a ready
// [Client]. Call [Client.SignIn] to authenticate.
func Dial(addr string, opts Options) (*Client, error) {
	d := net.Dialer{Timeout: opts.DialTimeout}
	conn, err := d.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("toc: dial %s: %w", addr, err)
	}
	return New(conn, opts), nil
}

// ScreenName returns the screen name supplied to [Client.SignIn], or the empty
// string before sign-in completes.
func (c *Client) ScreenName() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.screenNm
}

// SignIn performs the TOC sign-on handshake: sends "FLAPON", exchanges signon
// FLAP frames, sends the toc_signon command with a roasted password, waits for
// SIGN_ON (or ERROR), and finally sends toc_init_done to bring the session
// fully online. It must be called once before [Client.Receive].
func (c *Client) SignIn(screenName, password string) error {
	if _, err := c.conn.Write([]byte("FLAPON\n\n")); err != nil {
		return fmt.Errorf("toc: send FLAPON: %w", err)
	}

	serverSignon, err := readFrame(c.reader)
	if err != nil {
		return fmt.Errorf("toc: read server signon: %w", err)
	}
	if serverSignon.FrameType != FrameSignon {
		return fmt.Errorf("toc: expected signon frame, got type %#x", serverSignon.FrameType)
	}

	signonPayload := encodeSignonPayload(tlv{tag: screenNameTag, value: []byte(screenName)})
	if err := c.writeFrameLocked(FLAPFrame{FrameType: FrameSignon, Payload: signonPayload}); err != nil {
		return fmt.Errorf("toc: send client signon: %w", err)
	}

	roasted := RoastPassword([]byte(password))
	passHex := "0x" + hex.EncodeToString(roasted)
	signonCmd := fmt.Sprintf(`toc_signon "" "" %s %s`, screenName, passHex)
	if err := c.SendCommand(signonCmd); err != nil {
		return fmt.Errorf("toc: send toc_signon: %w", err)
	}

	for {
		msg, err := c.readData()
		if err != nil {
			return fmt.Errorf("toc: read signon reply: %w", err)
		}
		switch {
		case strings.HasPrefix(msg, "ERROR:"):
			return &SignInError{Code: strings.TrimPrefix(msg, "ERROR:")}
		case strings.HasPrefix(msg, "SIGN_ON:"):
			c.mu.Lock()
			c.online = true
			c.screenNm = screenName
			c.mu.Unlock()
			return c.SendCommand("toc_init_done")
		default:
			// CONFIG/NICK may precede or follow SIGN_ON depending on ordering;
			// keep reading until we see one of the terminating messages.
		}
	}
}

// SendCommand writes an arbitrary TOC command string as a FLAP data frame. Most
// callers should use the typed helpers ([Client.SendIM], [Client.SetAway]);
// SendCommand is exposed for commands without a dedicated method.
func (c *Client) SendCommand(cmd string) error {
	return c.writeFrameLocked(FLAPFrame{FrameType: FrameData, Payload: []byte(cmd)})
}

// SendIM sends an instant message to the user named "to". The message text is
// escaped and quoted per the TOC protocol.
func (c *Client) SendIM(to, text string) error {
	return c.SendCommand(fmt.Sprintf("toc_send_im %s %s", to, quote(text)))
}

// SetAway sets or clears the away message. A non-empty msg marks the session
// unavailable with the given (basic HTML) message; an empty msg marks the
// session available again.
func (c *Client) SetAway(msg string) error {
	if msg == "" {
		return c.SendCommand("toc_set_away")
	}
	return c.SendCommand(fmt.Sprintf("toc_set_away %s", quote(msg)))
}

// SetInfo sets the user profile / info string (basic HTML).
func (c *Client) SetInfo(info string) error {
	return c.SendCommand(fmt.Sprintf("toc_set_info %s", quote(info)))
}

// AddBuddy adds one or more screen names to the active buddy list.
func (c *Client) AddBuddy(names ...string) error {
	if len(names) == 0 {
		return nil
	}
	return c.SendCommand("toc_add_buddy " + strings.Join(names, " "))
}

// Receive runs the receive loop, decoding server messages and dispatching them
// to the configured [Handler] and/or [Options.OnEvent] until ctx is canceled,
// the connection closes, or a fatal read error occurs. It blocks the calling
// goroutine; run it in its own goroutine if you need to send commands
// concurrently. When KeepAlive is configured, keep-alive frames are sent from
// this method's lifecycle.
func (c *Client) Receive(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if c.keepAlive > 0 {
		go c.keepAliveLoop(ctx)
	}

	// Closing the connection on context cancellation unblocks a pending Read.
	// shutdown (not Close) is used so cancellation does not block on a signoff
	// write that the peer may never read.
	go func() {
		<-ctx.Done()
		c.shutdown()
	}()

	for {
		msg, err := c.readData()
		if err != nil {
			return err
		}
		c.dispatch(msg)
	}
}

// Close sends a best-effort signoff frame and closes the underlying connection.
// It is safe to call multiple times. If the connection is closed due to context
// cancellation during [Client.Receive], a later call to Close is a no-op.
func (c *Client) Close() error {
	c.closeOnce.Do(func() {
		c.markClosed()
		c.bestEffortSignoff()
		c.closeErr = c.conn.Close()
	})
	return c.closeErr
}

// shutdown closes the connection immediately without writing a signoff frame.
// It is used to unblock [Client.Receive] on context cancellation and is
// idempotent with [Client.Close].
func (c *Client) shutdown() {
	c.closeOnce.Do(func() {
		c.markClosed()
		c.closeErr = c.conn.Close()
	})
}

// markClosed records that the connection is closing so further writes are
// rejected.
func (c *Client) markClosed() {
	c.connMu.Lock()
	c.closed = true
	c.connMu.Unlock()
}

// bestEffortSignoff attempts to write a signoff frame, bounding the wait so it
// never blocks when the peer is not reading.
func (c *Client) bestEffortSignoff() {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(time.Second))
	_ = writeFrame(c.conn, FLAPFrame{FrameType: FrameSignoff, Sequence: c.seq})
	c.seq++
	_ = c.conn.SetWriteDeadline(time.Time{})
}

// dispatch decodes and delivers a message to the configured handlers.
func (c *Client) dispatch(msg string) {
	ev := parseEvent(msg)
	if c.onEvent != nil {
		c.onEvent(c, ev)
	}
	if c.handler != nil {
		switch ev.Type {
		case EventIM:
			c.handler.OnIM(ev.From, ev.Text, ev.Auto)
		case EventError:
			c.handler.OnError(ev.Code)
		}
	}
}

// keepAliveLoop sends keep-alive frames at the configured interval until ctx
// is done or sending fails.
func (c *Client) keepAliveLoop(ctx context.Context) {
	t := time.NewTicker(c.keepAlive)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := c.writeFrameLocked(FLAPFrame{FrameType: FrameKeepAlive}); err != nil {
				return
			}
		}
	}
}

// writeFrameLocked assigns the next sequence number, writes the frame, and
// returns any error.
func (c *Client) writeFrameLocked(f FLAPFrame) error {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	if c.closed {
		return net.ErrClosed
	}
	f.Sequence = c.seq
	c.seq++
	return writeFrame(c.conn, f)
}

// readData reads the next data frame, returning its payload as a string with
// trailing NUL bytes trimmed. Non-data frames are handled: keep-alive frames
// are skipped, signoff frames yield io.EOF, and signon frames are skipped.
func (c *Client) readData() (string, error) {
	for {
		f, err := readFrame(c.reader)
		if err != nil {
			return "", err
		}
		switch f.FrameType {
		case FrameKeepAlive, FrameSignon, FrameError:
			continue
		case FrameSignoff:
			return "", io.EOF
		case FrameData:
			return string(trimTrailingNUL(f.Payload)), nil
		default:
			continue
		}
	}
}

// trimTrailingNUL removes trailing zero bytes, mirroring the server which
// trims the NUL terminator some clients append.
func trimTrailingNUL(b []byte) []byte {
	for len(b) > 0 && b[len(b)-1] == 0 {
		b = b[:len(b)-1]
	}
	return b
}
