package network

const (
	NetFlagAllowStateless = 1

	NetSendFlagVital    = 1
	NetSendFlagConnless = 2
	NetSendFlagFlush    = 4

	NetStateOffline    = 0
	NetStateConnecting = 1
	NetStateOnline     = 2

	NetBanTypeSoft = 1
	NetBanTypeDrop = 2

	NetCreateFlagRandomPort = 1
)

const (
	NetMaxChunkHeaderSize = 3

	// packets
	NetPacketHeaderSize         = 7
	NetPacketHeaderSizeConnless = NetPacketHeaderSize + 2
	NetMaxPacketHeaderSize      = NetPacketHeaderSizeConnless
	//
	NetMaxPacketsize = 1400
	NetMaxPayload    = NetMaxPacketsize - NetMaxPacketHeaderSize

	NetPacketversion = 1

	NetPacketFlagControl     = 1
	NetPacketFlagResend      = 2
	NetPacketFlagCompression = 4
	NetPacketFlagConnless    = 8

	NetMaxPacketChunks = 256

	// token
	NetSeedTime = 16

	NetTokenCacheSize          = 64
	NetTokenCacheAddressexpiry = NetSeedTime
	NetTokenCachePacketexpiry  = 5
)

const (
	NetTokenMax  = 0xffffffff
	NetTokenNone = NetTokenMax
	NetTokenMask = NetTokenMax
)

const (
	NetTokenFlagAllowbroadcast = 1
	NetTokenFlagResponseonly   = 2

	NetTokenRequestDatasize = 512

	NetMaxClients        = 64
	NetMaxConsoleClients = 4

	NetMaxSequence  = 1 << 10
	NetSequenceMask = NetMaxSequence - 1

	NetConnStateOffline = 0
	NetConnStateToken   = 1
	NetConnStateConnect = 2
	NetConnStatePending = 3
	NetConnStateOnline  = 4
	NetConnStateError   = 5

	NetChunkFlagVital  = 1
	NetChunkFlagResend = 2

	NetCtrlMsgKeepAlive = 0
	NetCtrlMsgConnect   = 1
	NetCtrlMsgAccept    = 2
	NetCtrlMsgClose     = 4
	NetCtrlMsgToken     = 5

	NetConnBufferSize = 1024 * 32

	NetEnumTerminator = 1024*32 + 1
)
