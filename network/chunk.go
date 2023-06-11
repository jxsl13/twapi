package network

import (
	"errors"
	"fmt"
	"time"

	"github.com/jxsl13/twapi/protocol"
)

var (
	ErrChunkHeaderDataTooSmall = errors.New("chunk header data too small")
)

type Chunk struct {
	// -1 means that it's a connless packet
	// 0 on the client means the server
	ClientID int
	Addr     NetAddr // only used when cid == -1

	Flags protocol.SendFlags
	Data  []byte
}

type ChunkResend struct {
	Flags protocol.ChunkFlags
	Data  []byte

	Sequence      int
	LastSendTime  time.Time
	FirstSendTime time.Time
}

type ChunkHeader struct {
	Flags    protocol.ChunkFlags
	Size     int // TODO: do I need the size here
	Sequence int
}

func (h *ChunkHeader) IsVital() bool {
	return h.Flags&protocol.NetChunkFlagVital != 0
}

// Pack adds the header to the provided buffer
// The passed buffer must have preallocated space
// accessible via index
func (h *ChunkHeader) Pack(buffer []byte) []byte {
	// bounds check
	_ = buffer[protocol.NetMaxChunkHeaderSize-1]

	buffer[0] = byte((h.Flags&0x03)<<6) | byte(h.Size>>6)&0x3F
	buffer[1] = byte(h.Size & 0x3F)
	if h.IsVital() {
		buffer[1] |= byte(h.Sequence >> 2 & 0xC0)
		buffer[2] = byte(h.Sequence & 0xFF)
		return buffer[3:]
	}

	return buffer[2:]
}

// Unpack extracts the chunk header from the passed data bytes.
// The returned byte slice points to the not yet consumed bytes
// skipping those
func (h *ChunkHeader) Unpack(data []byte) ([]byte, error) {
	if len(data) < protocol.NetMaxChunkHeaderSize {
		return data, fmt.Errorf("%w: %d", ErrChunkHeaderDataTooSmall, len(data))
	}
	// _ = data[protocol.NetMaxChunkHeaderSize-1] // bounds check

	h.Flags = protocol.ChunkFlags((data[0] >> 6) & 0x03)
	h.Size = int(((data[0] & 0x3F) << 6) | (data[1] & 0x3F))
	h.Sequence = -1

	if h.IsVital() {
		h.Sequence = int(((data[1] & 0xC0) << 2) | data[2])
		return data[3:], nil
	}

	return data[2:], nil
}
