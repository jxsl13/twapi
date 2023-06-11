package network

import (
	"time"

	"github.com/jxsl13/twapi/protocol"
)

type Conn struct {
	state protocol.ConnState

	sequence int
	ack      int
	peerAck  int

	remoteClosed  int
	blockCloseMsg bool

	// TODO: ringbuffer
	buffer []ChunkResend

	lastUpdateTime time.Time
	lastRecvTime   time.Time
	lastSendTime   time.Time

	err error

	construct PacketConstruct

	token     protocol.Token
	peerToken protocol.Token
	peerAddr  NetAddr

	stats NetStats
	base  *NetBase
}

func NewConn(base *NetBase, blockCloseMsg bool) Conn {
	conn := Conn{
		base:          base,
		blockCloseMsg: blockCloseMsg,
		buffer:        make([]ChunkResend, protocol.NetConnBufferSize),
	}

	return conn
}

func (c *Conn) AckOne() {
	c.ack = (c.ack + 1) % protocol.NetMaxSequence
}

func (c *Conn) AckSequence() int {
	return c.ack
}

func (c *Conn) NextAck() int {
	return (c.ack + 1) % protocol.NetMaxSequence
}

func (c *Conn) SignalResend() {
	c.construct.Flags |= protocol.NetPacketFlagResend
}

func (c *Conn) reset() {
	c.sequence = 0
	c.ack = 0
	c.peerAck = 0
	c.remoteClosed = 0

	c.state = protocol.ConnStateOffline
	var zero time.Time

	c.lastSendTime = zero
	c.lastRecvTime = zero
	c.lastUpdateTime = zero
	c.token = protocol.NetTokenNone
	c.peerToken = protocol.NetTokenNone
	c.peerAddr = NilNetAddr

	c.buffer = make([]ChunkResend, protocol.NetConnBufferSize)

	var construct PacketConstruct
	c.construct = construct
}

func (c *Conn) resetStats() {
	var stats NetStats
	c.stats = stats
}

func (c *Conn) setError(err error) {
	c.err = err
}

func (c *Conn) ackChunks(ack int) {

	for _, resend := range c.buffer {
		if c.IsSeqInBackroom(resend.Sequence, ack) {
			c.buffer = c.buffer[1:]
		} else {
			break
		}
	}
}

func (c *Conn) queueChunkEx(flags protocol.ChunkFlags, sequence int, data []byte) int {
	return 0
}

// TODO: make controlMsg a typed enum
func (c *Conn) sendControl(controlMsg int, extraData ...byte) error {
	return nil
}

// TODO: make controlMsg a typed enum
func (c *Conn) sendControlWithToken(controlMsg int) error {
	return nil
}

func (c *Conn) resendChunk(resend ChunkResend) error {
	return nil
}

func (c *Conn) resend() error {
	return nil
}

func (c *Conn) generateToken(addr NetAddr) protocol.Token {
	return protocol.NetTokenNone
}

func (c *Conn) IsSeqInBackroom(seq, ack int) bool {
	bottom := (ack - protocol.NetMaxSequence/2)
	if bottom < 0 {
		if seq <= ack {
			return true
		}
		if seq >= (bottom + protocol.NetMaxSequence) {
			return true

		}
	} else if seq <= ack && seq >= bottom {
		return true
	}

	return false
}
