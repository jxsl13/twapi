package protocol

const (
	NetPacketHeaderSize         = 7
	NetPacketHeaderSizeConnless = NetPacketHeaderSize + 2
	NetMaxPacketHeaderSize      = NetPacketHeaderSizeConnless

	NetMaxPacketSize = 1400
	NetMaxPayload    = NetMaxPacketSize - NetMaxPacketHeaderSize

	NetPacketVersion   = 1
	NetMaxPacketChunks = 256
)

const (
	NetPacketFlagControl     PacketFlags = 1
	NetPacketFlagResend      PacketFlags = 2
	NetPacketFlagCompression PacketFlags = 4
	NetPacketFlagConnless    PacketFlags = 8
)

type PacketFlags int
