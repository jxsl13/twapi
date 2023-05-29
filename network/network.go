package network

import (
	"time"

	"github.com/jxsl13/twapi/protocol"
)

const (
	// token
	NetTokenCacheAddressExpiry = protocol.NetSeedTime
	NetTokenCachePacketExpiry  = 5 * time.Second

	NetTokenCacheSize = 64
)

const (
	NetMaxChunkHeaderSize = 3

	// packets
	NetPacketHeaderSize         = 7
	NetPacketHeaderSizeConnless = NetPacketHeaderSize + 2
	NetMaxPacketHeaderSize      = NetPacketHeaderSizeConnless

	NetMaxPacketsize = 1400
	NetMaxPayload    = NetMaxPacketsize - NetMaxPacketHeaderSize

	NetMaxPacketChunks = 256

	NetTokenRequestDataSize = 512

	NetConnBufferSize = 1024 * 32

	NetMaxClients        = 64
	NetMaxConsoleClients = 4

	NetMaxSequence  = 1 << 10
	NetSequenceMask = NetMaxSequence - 1
)
