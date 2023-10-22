package network

import "github.com/jxsl13/twapi/protocol"

var NilPacketContruct PacketConstruct

type PacketConstruct struct {
	Flags     protocol.PacketFlags
	Ack       int
	NumChunks int
	ChunkData []byte
}
