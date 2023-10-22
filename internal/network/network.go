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
