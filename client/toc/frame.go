package toc

import (
	"encoding/binary"
	"fmt"
	"io"
)

// FLAP frame types exchanged over a TOC/SFLAP connection.
const (
	// FrameSignon is the signon frame type (0x01), sent during the handshake.
	FrameSignon byte = 0x01
	// FrameData carries a TOC command string payload (0x02).
	FrameData byte = 0x02
	// FrameError is an error frame type (0x03).
	FrameError byte = 0x03
	// FrameSignoff indicates the peer is disconnecting (0x04).
	FrameSignoff byte = 0x04
	// FrameKeepAlive is a no-op heartbeat frame (0x05).
	FrameKeepAlive byte = 0x05
)

const (
	flapMarker    byte = 0x2A
	flapHeaderLen      = 6
	signonVersion      = uint32(1)
	// ScreenNameTag is the signon-frame TLV tag carrying the client screen
	// name (0x0001). It mirrors wire.LoginTLVTagsScreenName.
	screenNameTag uint16 = 0x0001
)

// FLAPFrame is a single SFLAP/FLAP frame exchanged between client and server.
//
// Wire layout (big-endian):
//
//	0x2A | frameType(1) | sequence(2) | payloadLen(2) | payload(payloadLen)
type FLAPFrame struct {
	FrameType byte
	Sequence  uint16
	Payload   []byte
}

// tlv is a tag-length-value pair used inside signon frame payloads.
//
// Wire layout (big-endian): tag(2) | len(2) | value(len).
type tlv struct {
	tag   uint16
	value []byte
}

// readFrame reads a single FLAP frame from r. It blocks until a full frame is
// available or an error (including io.EOF) occurs.
func readFrame(r io.Reader) (FLAPFrame, error) {
	header := make([]byte, flapHeaderLen)
	if _, err := io.ReadFull(r, header); err != nil {
		return FLAPFrame{}, err
	}
	if header[0] != flapMarker {
		return FLAPFrame{}, fmt.Errorf("toc: invalid FLAP start marker %#x", header[0])
	}
	f := FLAPFrame{
		FrameType: header[1],
		Sequence:  binary.BigEndian.Uint16(header[2:4]),
	}
	payloadLen := int(binary.BigEndian.Uint16(header[4:6]))
	if payloadLen < 0 {
		return FLAPFrame{}, fmt.Errorf("toc: negative FLAP payload length")
	}
	if payloadLen > 0 {
		f.Payload = make([]byte, payloadLen)
		if _, err := io.ReadFull(r, f.Payload); err != nil {
			return FLAPFrame{}, err
		}
	}
	return f, nil
}

// writeFrame writes a single FLAP frame to w.
func writeFrame(w io.Writer, f FLAPFrame) error {
	buf := make([]byte, flapHeaderLen+len(f.Payload))
	buf[0] = flapMarker
	buf[1] = f.FrameType
	binary.BigEndian.PutUint16(buf[2:4], f.Sequence)
	binary.BigEndian.PutUint16(buf[4:6], uint16(len(f.Payload)))
	copy(buf[6:], f.Payload)
	_, err := w.Write(buf)
	return err
}

// appendTLV appends the encoded form of t to buf and returns the result.
func appendTLV(buf []byte, t tlv) []byte {
	var head [4]byte
	binary.BigEndian.PutUint16(head[0:2], t.tag)
	binary.BigEndian.PutUint16(head[2:4], uint16(len(t.value)))
	buf = append(buf, head[:]...)
	buf = append(buf, t.value...)
	return buf
}

// encodeSignonPayload builds the payload for a FLAP signon frame: a 4-byte
// version field (0x00000001) followed by the given TLVs with no count prefix.
func encodeSignonPayload(tlvs ...tlv) []byte {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload[0:4], signonVersion)
	for _, t := range tlvs {
		payload = appendTLV(payload, t)
	}
	return payload
}

// parseSignonPayload parses a signon frame payload into a tag->value map. The
// leading 4-byte version field is consumed and discarded.
func parseSignonPayload(payload []byte) (map[uint16][]byte, error) {
	tlvs := make(map[uint16][]byte)
	if len(payload) < 4 {
		return nil, fmt.Errorf("toc: signon payload too short (%d bytes)", len(payload))
	}
	payload = payload[4:]
	for len(payload) >= 4 {
		tag := binary.BigEndian.Uint16(payload[0:2])
		l := int(binary.BigEndian.Uint16(payload[2:4]))
		payload = payload[4:]
		if len(payload) < l {
			return nil, fmt.Errorf("toc: truncated TLV %#x: want %d bytes, have %d", tag, l, len(payload))
		}
		val := make([]byte, l)
		copy(val, payload[:l])
		tlvs[tag] = val
		payload = payload[l:]
	}
	if len(payload) != 0 {
		return nil, fmt.Errorf("toc: trailing %d bytes in signon payload", len(payload))
	}
	return tlvs, nil
}
