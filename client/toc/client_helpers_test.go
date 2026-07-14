package toc

import (
	"bufio"
	"io"
	"net"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// im captures a delivered instant message for recordingHandler.
type im struct {
	from string
	text string
	auto bool
}

// recordingHandler is a [Handler] that records delivered events for tests. It
// also signals each delivery through buffered channels so tests can wait
// deterministically.
type recordingHandler struct {
	mu    sync.Mutex
	ims   []im
	errs  []string
	imCh  chan im
	errCh chan string
}

func newRecordingHandler() *recordingHandler {
	return &recordingHandler{
		imCh:  make(chan im, 8),
		errCh: make(chan string, 8),
	}
}

func (h *recordingHandler) OnIM(from, text string, auto bool) {
	h.mu.Lock()
	h.ims = append(h.ims, im{from, text, auto})
	h.mu.Unlock()
	select {
	case h.imCh <- im{from, text, auto}:
	default:
	}
}

func (h *recordingHandler) OnError(code string) {
	h.mu.Lock()
	h.errs = append(h.errs, code)
	h.mu.Unlock()
	select {
	case h.errCh <- code:
	default:
	}
}

func (h *recordingHandler) snapshot() ([]im, []string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	ims := make([]im, len(h.ims))
	copy(ims, h.ims)
	errs := make([]string, len(h.errs))
	copy(errs, h.errs)
	return ims, errs
}

// fakeServer drives the server side of a TOC connection over a net.Pipe. It
// mirrors the handshake performed by server/toc (FLAPON, signon frame exchange,
// toc_signon, SIGN_ON reply, toc_init_done).
type fakeServer struct {
	t    *testing.T
	conn net.Conn
	r    *bufio.Reader
}

// newFakeServer returns a client backed by an in-memory pipe and the matching
// server-side driver.
func newFakeServer(t *testing.T) (*Client, *fakeServer) {
	t.Helper()
	clientConn, serverConn := net.Pipe()
	fs := &fakeServer{t: t, conn: serverConn, r: bufio.NewReader(serverConn)}
	return New(clientConn, Options{}), fs
}

func (fs *fakeServer) expectFLAPON() {
	fs.t.Helper()
	b := make([]byte, 8)
	_, err := io.ReadFull(fs.r, b)
	assert.NoError(fs.t, err)
	assert.Equal(fs.t, "FLAPON\n\n", string(b))
}

// sendSignonFrame sends an empty server signon frame (version 1, no TLVs),
// matching server/toc/server.go initFLAP.
func (fs *fakeServer) sendSignonFrame() {
	fs.t.Helper()
	err := writeFrame(fs.conn, FLAPFrame{FrameType: FrameSignon, Payload: encodeSignonPayload()})
	assert.NoError(fs.t, err)
}

func (fs *fakeServer) readFrame() FLAPFrame {
	fs.t.Helper()
	f, err := readFrame(fs.r)
	assert.NoError(fs.t, err)
	return f
}

func (fs *fakeServer) readData() string {
	fs.t.Helper()
	for {
		f := fs.readFrame()
		if f.FrameType == FrameData {
			return string(f.Payload)
		}
	}
}

func (fs *fakeServer) sendData(payload string) {
	fs.t.Helper()
	err := writeFrame(fs.conn, FLAPFrame{FrameType: FrameData, Payload: []byte(payload)})
	assert.NoError(fs.t, err)
}

// handshake performs the sign-on handshake through the toc_signon command,
// returning the toc_signon command string the client sent.
func (fs *fakeServer) handshake(expectedScreenName string) string {
	fs.t.Helper()
	fs.expectFLAPON()
	fs.sendSignonFrame()
	signon := fs.readFrame()
	assert.Equal(fs.t, FrameSignon, signon.FrameType)
	tlvs, err := parseSignonPayload(signon.Payload)
	assert.NoError(fs.t, err)
	assert.Equal(fs.t, []byte(expectedScreenName), tlvs[screenNameTag])
	return fs.readData()
}

// acceptOnline completes the handshake by sending SIGN_ON and reading
// toc_init_done.
func (fs *fakeServer) acceptOnline() {
	fs.t.Helper()
	fs.sendData("SIGN_ON:TOC1.0")
	assert.Equal(fs.t, "toc_init_done", fs.readData())
}
