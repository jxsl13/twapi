package network

type NetChunkHeader struct {
	Flags    int
	Size     int
	Sequence int
}

// TODO: implement
func (nch *NetChunkHeader) Pack(data []byte) byte {
	return 0
}

func (nch *NetChunkHeader) Unpack(data []byte) byte {
	return 0
}
