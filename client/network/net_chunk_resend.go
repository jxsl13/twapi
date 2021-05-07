package network

type NetChunkResend struct {
	Flags    int
	DataSize int
	Data     []byte

	Sequence      int
	LastSendTime  int64
	FirstSendTime int64
}
