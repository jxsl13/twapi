package protocol

const (
	NetMaxChunkHeaderSize = 3

	NetChunkFlagVital  ChunkFlags = 1
	NetChunkFlagResend ChunkFlags = 2
)

type ChunkFlags int
