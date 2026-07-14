package toc

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestWriteFrame_KnownVectors asserts the on-wire FLAP byte layout matches the
// format the server reads (see wire/frames.go FLAPFrame):
//
//	0x2A | type(1) | seq(2 BE) | len(2 BE) | payload(len)
func TestWriteFrame_KnownVectors(t *testing.T) {
	tests := []struct {
		name string
		f    FLAPFrame
		want []byte
	}{
		{
			name: "data frame with payload",
			f:    FLAPFrame{FrameType: FrameData, Sequence: 0x0001, Payload: []byte("hi")},
			want: []byte{0x2A, 0x02, 0x00, 0x01, 0x00, 0x02, 'h', 'i'},
		},
		{
			name: "keepalive frame no payload",
			f:    FLAPFrame{FrameType: FrameKeepAlive, Sequence: 0x0005},
			want: []byte{0x2A, 0x05, 0x00, 0x05, 0x00, 0x00},
		},
		{
			name: "signoff frame no payload",
			f:    FLAPFrame{FrameType: FrameSignoff, Sequence: 0x0002},
			want: []byte{0x2A, 0x04, 0x00, 0x02, 0x00, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			assert.NoError(t, writeFrame(&buf, tt.f))
			assert.Equal(t, tt.want, buf.Bytes())
		})
	}
}

// TestReadFrame_RoundTrip verifies readFrame decodes what writeFrame encodes,
// including the sequence number and payload.
func TestReadFrame_RoundTrip(t *testing.T) {
	frames := []FLAPFrame{
		{FrameType: FrameData, Sequence: 1, Payload: []byte("toc_init_done")},
		{FrameType: FrameKeepAlive, Sequence: 2},
		{FrameType: FrameSignon, Sequence: 3, Payload: []byte{0x00, 0x00, 0x00, 0x01}},
	}

	var buf bytes.Buffer
	for _, f := range frames {
		assert.NoError(t, writeFrame(&buf, f))
	}

	for _, want := range frames {
		got, err := readFrame(&buf)
		assert.NoError(t, err)
		assert.Equal(t, want.FrameType, got.FrameType)
		assert.Equal(t, want.Sequence, got.Sequence)
		assert.Equal(t, want.Payload, got.Payload)
	}
}

func TestReadFrame_InvalidMarker(t *testing.T) {
	_, err := readFrame(bytes.NewReader([]byte{0x00, 0x02, 0x00, 0x00, 0x00, 0x00}))
	assert.Error(t, err)
}

// TestEncodeSignonPayload verifies the signon frame payload layout:
// version(4 BE) followed by TLVs (tag/len/value) with no count prefix. This is
// what the client sends during initFLAP and what the server reads via
// wire.FlapClient.ReceiveSignonFrame.
func TestEncodeSignonPayload(t *testing.T) {
	got := encodeSignonPayload(tlv{tag: screenNameTag, value: []byte("me")})
	want := []byte{
		0x00, 0x00, 0x00, 0x01, // FLAP version 1
		0x00, 0x01, // TLV tag (screen name)
		0x00, 0x02, // TLV length
		'm', 'e',
	}
	assert.Equal(t, want, got)
}

// TestParseSignonPayload verifies decoding the server's signon frame and that
// the screen name TLV (0x0001) round-trips.
func TestParseSignonPayload(t *testing.T) {
	payload := encodeSignonPayload(
		tlv{tag: screenNameTag, value: []byte("testuser")},
		tlv{tag: 0x0017, value: []byte{0xAB, 0xCD}},
	)
	tlvs, err := parseSignonPayload(payload)
	assert.NoError(t, err)
	assert.Equal(t, []byte("testuser"), tlvs[screenNameTag])
	assert.Equal(t, []byte{0xAB, 0xCD}, tlvs[0x0017])
}

func TestParseSignonPayload_Errors(t *testing.T) {
	t.Run("too short", func(t *testing.T) {
		_, err := parseSignonPayload([]byte{0x00, 0x00})
		assert.Error(t, err)
	})
	t.Run("truncated TLV", func(t *testing.T) {
		_, err := parseSignonPayload([]byte{0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x05, 'a'})
		assert.Error(t, err)
	})
}
