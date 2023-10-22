package network

import (
	"errors"
	"fmt"

	"github.com/jxsl13/twapi/protocol"
)

var (
	ErrOffline       = errors.New("connection offline")
	ErrChunkOverflow = errors.New("chunk overflow")
	ErrDataExhausted = errors.New("data exhausted")
)

type NetRecvUnpacker struct {
	active bool

	conn         *Conn
	addr         NetAddr
	currentChunk int
	clientID     int

	data   PacketConstruct
	buffer [protocol.NetMaxPacketSize]byte
}

func NewRecvUnpacker() NetRecvUnpacker {
	return NetRecvUnpacker{
		active:       true,
		conn:         nil,
		currentChunk: 0,
		clientID:     0,
	}
}

func (u *NetRecvUnpacker) Deactivate() {
	u.active = false
}

func (u *NetRecvUnpacker) IsActive() bool {
	return u.active
}

func (u *NetRecvUnpacker) Start(addr NetAddr, conn *Conn, clientID int) {
	u.addr = addr
	u.conn = conn
	u.clientID = clientID
}

func (u *NetRecvUnpacker) FetchChunk() (chunk Chunk, err error) {

	if u.conn.state == protocol.ConnStateOffline {
		u.Deactivate()
		return chunk, fmt.Errorf("failed to fetch chunk: %w", ErrOffline)
	}

	var header ChunkHeader

	for {
		data := u.data.ChunkData
		if !u.IsActive() || u.currentChunk >= u.data.NumChunks {
			u.Deactivate()
			return chunk, fmt.Errorf("failed to fetch chunk: %w", ErrChunkOverflow)
		}

		// walk over data to the current chunk
		// must be done from first until current chunk in order to skip
		// chunk data correctly
		for i := 0; i < u.currentChunk; i++ {
			data, err = header.Unpack(data)
			if err != nil {
				return chunk, fmt.Errorf("failed to fetch chunk: %w", err)
			}
			data = data[header.Size:]
		}
		data, err = header.Unpack(data)
		if err != nil {
			return chunk, fmt.Errorf("failed to fetch chunk: invalid current chunk: %w", err)
		}
		u.currentChunk++
		if len(data) < header.Size {
			u.Deactivate()
			return chunk, fmt.Errorf("failed to fetch chunk: %w", ErrDataExhausted)
		}

		if header.IsVital() {
			if header.Sequence == u.conn.NextAck() {
				u.conn.AckOne()
			} else if u.conn.IsSeqInBackroom(header.Sequence, u.conn.AckSequence()) {
				continue
			} else {
				u.conn.SignalResend() // request resend
				continue              // take the next chunk in the packet
			}
		}

		var flags protocol.SendFlags
		if header.IsVital() {
			flags = protocol.NetSendFlagVital
		}

		chunkData := make([]byte, header.Size)
		copy(chunkData, data[:header.Size])

		chunk = Chunk{
			ClientID: u.clientID,
			Addr:     u.conn.peerAddr,
			Flags:    flags,
			Data:     chunkData,
		}

		return chunk, nil

	}
}

/*
class CNetRecvUnpacker
{
	bool m_Valid;

public:
	NETADDR m_Addr;
	CNetConnection *m_pConnection;
	int m_CurrentChunk;
	int m_ClientID;
	CNetPacketConstruct m_Data;
	unsigned char m_aBuffer[NET_MAX_PACKETSIZE];

	CNetRecvUnpacker() { Clear(); }
	bool IsActive() { return m_Valid; }
	void Clear();
	void Start(const NETADDR *pAddr, CNetConnection *pConnection, int ClientID);
	int FetchChunk(CNetChunk *pChunk);
};

*/
