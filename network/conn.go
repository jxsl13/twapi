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
	// TStaticRingBuffer<CNetChunkResend, NET_CONN_BUFFERSIZE> m_Buffer;

	lastUpdateTime time.Time
	lastRecvTime   time.Time
	lastSendTime   time.Time

	err error

	construct PacketConstruct

	token     protocol.Token
	peerToken protocol.Token
	peerAddr  NetAddr

	stats NetStats
	//CNetBase *m_pNetBase;

}

func NewConn(blockCloseMsg bool) *Conn {
	conn := Conn{
		blockCloseMsg: blockCloseMsg,
	}

	// TODO: net conn
	// m_pNetBase = pNetBase;
	return &conn
}

func (c *Conn) Reset() {
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

	c.buffer = make([]ChunkResend, 0, NetConnBufferSize)

	var construct PacketConstruct
	c.construct = construct
}

func (c *Conn) ResetStats() {
	var stats NetStats
	c.stats = stats
}

func (c *Conn) SetError(err error) {
	c.err = err
}

func (c *Conn) AckChunks(ack int) {

	for _, resend := range c.buffer {
		if c.IsSeqInBackroom(resend.Sequence, ack) {
			c.buffer = c.buffer[1:]
		} else {
			break
		}
	}
}

// TODO: make flags typed
func (c *Conn) QueueChunkEx(flags int, sequence int, data []byte) int {
	return 0
}

// TODO: make controlMsg a typed enum
func (c *Conn) SendControl(controlMsg int, extraData ...byte) error {
	return nil
}

// TODO: make controlMsg a typed enum
func (c *Conn) SendControlWithToken(controlMsg int) error {
	return nil
}

func (c *Conn) ResendChunk(resend ChunkResend) error {
	return nil
}

func (c *Conn) Resend() error {
	return nil
}

func (c *Conn) GenerateToken(addr NetAddr) protocol.Token {
	return protocol.NetTokenNone
}

func (c *Conn) IsSeqInBackroom(seq, ack int) bool {
	bottom := (ack - NetMaxSequence/2)
	if bottom < 0 {
		if seq <= ack {
			return true
		}
		if seq >= (bottom + NetMaxSequence) {
			return true

		}
	} else if seq <= ack && seq >= bottom {
		return true
	}

	return false
}
