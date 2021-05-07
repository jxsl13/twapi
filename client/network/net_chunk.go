package network

import "net"

type NetChunk struct {
	ClientID int
	Address  net.UDPAddr
	Flags    int
	DataSize int
	Data     []byte
}
