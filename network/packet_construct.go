package network

var NilPacketContruct PacketConstruct

type PacketConstruct struct {
	Flags     int
	Ack       int
	NumChunks int
	ChunkData []byte
}
