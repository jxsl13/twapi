package network

import "inet.af/netaddr"

type NetChunk struct {
	ClientID int
	Address  netaddr.IPPort
	Flags    int
	DataSize int
	Data     []byte
}
