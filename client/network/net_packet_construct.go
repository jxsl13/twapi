package network

type NetPacketConstruct struct {
	Token         Token
	ResponseToken Token

	Flags     int
	Ack       int
	NumChunks int
	DataSize  int
	ChunkData [NetMaxPayload]byte
}
