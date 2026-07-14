package toc

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestSignIn_Success drives the full server handshake and asserts that SignIn
// emits a toc_signon with a roasted password and finishes by sending
// toc_init_done, matching the flow in server/toc/server.go dispatchFLAP.
func TestSignIn_Success(t *testing.T) {
	c, fs := newFakeServer(t)
	defer c.Close()
	defer fs.conn.Close()

	var signonCmd string
	done := make(chan struct{})
	go func() {
		defer close(done)
		signonCmd = fs.handshake("testuser")
		fs.acceptOnline()
	}()

	assert.NoError(t, c.SignIn("testuser", "secret"))
	<-done

	roasted := RoastPassword([]byte("secret"))
	want := fmt.Sprintf(`toc_signon "" "" testuser 0x%s`, hex.EncodeToString(roasted))
	assert.Equal(t, want, signonCmd)
	assert.Equal(t, "testuser", c.ScreenName())
}

// TestSignIn_AuthError asserts that an ERROR reply surfaces as *SignInError.
func TestSignIn_AuthError(t *testing.T) {
	c, fs := newFakeServer(t)
	defer c.Close()
	defer fs.conn.Close()

	go func() {
		fs.handshake("testuser")
		fs.sendData("ERROR:980")
	}()

	err := c.SignIn("testuser", "secret")
	var signInErr *SignInError
	if assert.ErrorAs(t, err, &signInErr) {
		assert.Equal(t, "980", signInErr.Code)
	}
}

// TestSendIM verifies the on-wire toc_send_im command string, including TOC
// escaping of the message body.
func TestSendIM(t *testing.T) {
	c, fs := newFakeServer(t)
	defer c.Close()
	defer fs.conn.Close()

	got := make(chan string, 1)
	go func() { got <- fs.readData() }()

	assert.NoError(t, c.SendIM("buddy", "hi there :)"))
	select {
	case cmd := <-got:
		assert.Equal(t, `toc_send_im buddy "hi there :\)"`, cmd)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for toc_send_im")
	}
}

// TestSetAway verifies both setting an away message and clearing it (returning
// online).
func TestSetAway(t *testing.T) {
	t.Run("away message", func(t *testing.T) {
		c, fs := newFakeServer(t)
		defer c.Close()
		defer fs.conn.Close()

		got := make(chan string, 1)
		go func() { got <- fs.readData() }()

		assert.NoError(t, c.SetAway("back soon"))
		select {
		case cmd := <-got:
			assert.Equal(t, `toc_set_away "back soon"`, cmd)
		case <-time.After(time.Second):
			t.Fatal("timeout")
		}
	})

	t.Run("online clears away", func(t *testing.T) {
		c, fs := newFakeServer(t)
		defer c.Close()
		defer fs.conn.Close()

		got := make(chan string, 1)
		go func() { got <- fs.readData() }()

		assert.NoError(t, c.SetAway(""))
		select {
		case cmd := <-got:
			assert.Equal(t, `toc_set_away`, cmd)
		case <-time.After(time.Second):
			t.Fatal("timeout")
		}
	})
}

// TestSendCommand verifies arbitrary command framing.
func TestSendCommand(t *testing.T) {
	c, fs := newFakeServer(t)
	defer c.Close()
	defer fs.conn.Close()

	got := make(chan string, 1)
	go func() { got <- fs.readData() }()

	assert.NoError(t, c.SendCommand("toc_set_idle 0"))
	select {
	case cmd := <-got:
		assert.Equal(t, "toc_set_idle 0", cmd)
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

// TestReceive_IM asserts the receive loop decodes IM_IN frames and dispatches
// them to the Handler.
func TestReceive_IM(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	h := newRecordingHandler()
	c := New(clientConn, Options{Handler: h})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	recvDone := make(chan struct{})
	go func() { _ = c.Receive(ctx); close(recvDone) }()

	assert.NoError(t, writeFrame(serverConn, FLAPFrame{FrameType: FrameData, Payload: []byte("IM_IN:alice:F:hello world")}))
	waitFor(t, h.imCh)

	ims, _ := h.snapshot()
	assert.Equal(t, []im{{from: "alice", text: "hello world", auto: false}}, ims)

	cancel()
	<-recvDone
}

// TestReceive_IM_IN2 asserts TOC2 IM_IN2 frames decode with the whisper field
// skipped.
func TestReceive_IM_IN2(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	h := newRecordingHandler()
	c := New(clientConn, Options{Handler: h})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	recvDone := make(chan struct{})
	go func() { _ = c.Receive(ctx); close(recvDone) }()

	assert.NoError(t, writeFrame(serverConn, FLAPFrame{FrameType: FrameData, Payload: []byte("IM_IN2:carol:T:F:yo")}))
	waitFor(t, h.imCh)

	ims, _ := h.snapshot()
	assert.Equal(t, []im{{from: "carol", text: "yo", auto: true}}, ims)

	cancel()
	<-recvDone
}

// TestReceive_Error asserts ERROR frames dispatch to OnError.
func TestReceive_Error(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	h := newRecordingHandler()
	c := New(clientConn, Options{Handler: h})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	recvDone := make(chan struct{})
	go func() { _ = c.Receive(ctx); close(recvDone) }()

	assert.NoError(t, writeFrame(serverConn, FLAPFrame{FrameType: FrameData, Payload: []byte("ERROR:901")}))
	waitFor(t, h.errCh)

	_, errs := h.snapshot()
	assert.Equal(t, []string{"901"}, errs)

	cancel()
	<-recvDone
}

// TestReceive_OnEvent asserts the OnEvent callback receives the parsed event
// and the client reference.
func TestReceive_OnEvent(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	events := make(chan Event, 4)
	var c *Client
	c = New(clientConn, Options{
		OnEvent: func(client *Client, ev Event) {
			assert.Equal(t, c, client)
			select {
			case events <- ev:
			default:
			}
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	recvDone := make(chan struct{})
	go func() { _ = c.Receive(ctx); close(recvDone) }()

	assert.NoError(t, writeFrame(serverConn, FLAPFrame{FrameType: FrameData, Payload: []byte("NICK:TestUser")}))
	select {
	case ev := <-events:
		assert.Equal(t, EventNick, ev.Type)
		assert.Equal(t, "TestUser", ev.Name)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}

	cancel()
	<-recvDone
}

// TestReceive_SignoffReturnsEOF asserts a signoff frame ends the receive loop.
func TestReceive_SignoffReturnsEOF(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	c := New(clientConn, Options{})

	done := make(chan error, 1)
	go func() { done <- c.Receive(context.Background()) }()

	assert.NoError(t, writeFrame(serverConn, FLAPFrame{FrameType: FrameSignoff}))

	select {
	case err := <-done:
		assert.ErrorIs(t, err, io.EOF)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for Receive to return")
	}
}

// TestReceive_KeepAliveSkipped asserts keepalive frames are silently skipped
// and do not surface as events or end the loop.
func TestReceive_KeepAliveSkipped(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	h := newRecordingHandler()
	c := New(clientConn, Options{Handler: h})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	recvDone := make(chan struct{})
	go func() {
		err := c.Receive(ctx)
		t.Logf("receive returned: %v", err)
		close(recvDone)
	}()

	assert.NoError(t, writeFrame(serverConn, FLAPFrame{FrameType: FrameKeepAlive}))
	assert.NoError(t, writeFrame(serverConn, FLAPFrame{FrameType: FrameData, Payload: []byte("IM_IN:dan:F:after keepalive")}))
	waitFor(t, h.imCh)

	ims, _ := h.snapshot()
	assert.Equal(t, []im{{from: "dan", text: "after keepalive", auto: false}}, ims)

	cancel()
	<-recvDone
}

// errEOFOrClosed returns a sentinel acceptable to assert.ErrorIs for the EOF /
// net.ErrClosed cases returned by Receive.
func errEOFOrClosed() error {
	return errors.Join(io.EOF, net.ErrClosed)
}

// waitFor blocks until ch yields a value or the test times out.
func waitFor[T any](t *testing.T, ch <-chan T) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for handler delivery")
	}
}
