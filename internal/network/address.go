package network

import (
	"net/netip"

	"github.com/jxsl13/twapi/protocol"
)

// NilNetAddr is the zero value of NetAddr
var NilNetAddr NetAddr

// ParseNetAddr parses a NetAddr from a string <ipv4/v6>:port representation
func ParseNetAddr(addrPort string, broadcast ...bool) (NetAddr, error) {
	br := false
	if len(broadcast) > 0 {
		br = broadcast[0]
	}

	ap, err := netip.ParseAddrPort(addrPort)
	if err != nil {
		return NilNetAddr, err
	}

	return NetAddr{
		broadcast: br,
		addr:      ap.Addr(),
		port:      ap.Port(),
	}, nil
}

// NetAddr represents a network address
// which is an ip, port and some extra information.
type NetAddr struct {
	addr      netip.Addr
	port      uint16
	broadcast bool // link broadcast in TW terms
}

// IsBroadcast returns true if the NetAddr type is a 'link broadcast' type.
func (n NetAddr) IsBroadcast() bool {
	return n.broadcast
}

func (n *NetAddr) SetBroadcast(broadcastAddr bool) {
	n.broadcast = broadcastAddr
}

func (n NetAddr) AddrPort() netip.AddrPort {
	return netip.AddrPortFrom(n.addr, n.port)
}

func (n NetAddr) Addr() netip.Addr {
	return n.addr
}

func (n *NetAddr) SetAddr(addr netip.Addr) {
	n.addr = addr
}

func (n NetAddr) Port() uint16 {
	return n.port
}

func (n *NetAddr) SetPort(port uint16) {
	n.port = port
}

// Type returns the protocol specific integer type
func (n NetAddr) Type() protocol.NetType {
	addr := n.addr

	if n.broadcast {
		if addr.Is4() || addr.Is4In6() {
			return protocol.NetTypeIPv4 | protocol.NetTypeLinkBroadcast
		}
		return protocol.NetTypeIPv6 | protocol.NetTypeLinkBroadcast

	} else if addr.Is4() || addr.Is4In6() {
		return protocol.NetTypeIPv4
	}
	return protocol.NetTypeIPv6
}

func (n NetAddr) IsReserved() bool {
	addr := n.addr
	return addr.IsUnspecified() ||
		addr.IsLoopback() ||
		addr.IsGlobalUnicast() ||
		addr.IsPrivate() ||
		addr.IsMulticast() ||
		addr.IsLinkLocalMulticast() ||
		addr.IsInterfaceLocalMulticast() ||
		addr.IsLinkLocalUnicast()
}

func (p NetAddr) MarshalBinary() ([]byte, error) {
	result := make([]byte, 0, 16+2+1+8) // add 8 bytes extra for
	iparr := p.addr.As16()
	result = append(result, iparr[:]...)

	// little endian
	port := p.port
	result = append(result, byte(port))
	result = append(result, byte(port>>8))

	if p.broadcast {
		result = append(result, 1)
	} else {
		result = append(result, 0)
	}
	return result, nil
}
