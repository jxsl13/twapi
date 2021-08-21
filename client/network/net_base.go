package network

import (
	"net"

	"github.com/jxsl13/twapi/compression"
	"inet.af/netaddr"
)

var (
	netInitializer = NewNetInitializer()
)

type NetInitializer struct {
}

func NewNetInitializer() NetInitializer {
	return NetInitializer{}
}

type NetBase struct {
	socket          net.UDPConn
	huffman         compression.Huffman
	requestTokenBuf [NetTokenRequestDataSize]byte
}

func (nb *NetBase) SendControlMsg(address netaddr.IPPort, token Token, Ack int, ControlMsg int, pExtra []byte, ExtraSize int) {

}

func (nb *NetBase) SendControlMsgWithToken(address netaddr.IPPort, token Token, Ack, ControlMsg int, MyToken Token, Extended bool) {

}
func (nb *NetBase) SendPacketConnless(address netaddr.IPPort, token Token, responseToken Token, pData []byte, DataSize int) {

}
func (nb *NetBase) SendPacket(address netaddr.IPPort, pPacket *NetPacketConstruct) {

}
func (nb *NetBase) UnpackPacket(address netaddr.IPPort, pBuffer []byte, pPacket *NetPacketConstruct) int {
	return 0
}
