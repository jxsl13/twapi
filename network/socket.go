package network

import (
	"math/rand"
	"net"
	"net/netip"
	"syscall"
)

// NilNetSocket is the zero value of an unitialized NetSocket.
var NilNetSocket NetSocket

type NetSocket struct {
	socket *net.UDPConn
}

func NewNetSocketFrom(bindAddr string, randomPort ...bool) (NetSocket, error) {
	ap, err := netip.ParseAddrPort(bindAddr)
	if err != nil {
		return NilNetSocket, err
	}

	return NewNetSocket(ap, randomPort...)
}

// NewSocket creates a new UDP socket for sending data to any IP.
// bindAddrPort expects an ip:port. In case the port is 0, the operating system will
// assign a random port.
// If you want a random hight port in the range between 49152 and 65535, then
// pass a true as addistional single extra parameter for 'randomPort'
func NewNetSocket(bindAddrPort netip.AddrPort, randomPort ...bool) (sock NetSocket, err error) {
	randPort := false
	if len(randomPort) > 0 {
		randPort = randomPort[0]
	} else if bindAddrPort.Port() == 0 {
		randPort = true
	}

	var conn *net.UDPConn
	const (
		portRange  = 16384
		maxRetries = portRange
	)

	addr := bindAddrPort.Addr()
	retries := 0
	for retries < maxRetries {
		if randPort {
			port := uint16(49152 + rand.Int31n(portRange)) // <= 65535
			laddr := net.UDPAddrFromAddrPort(netip.AddrPortFrom(addr, port))
			raddr := laddr // local and remote address are the same
			conn, err = net.DialUDP(laddr.Network(), laddr, raddr)
			if err != nil {
				retries++
				continue
			}
			break
		}
	}
	if err != nil {
		return NilNetSocket, err
	}

	defer func() {
		if err != nil {
			conn.Close()
		}
	}()

	const receiveSize = 65536
	err = conn.SetReadBuffer(receiveSize)
	if err != nil {
		return NilNetSocket, err
	}

	rc, err := conn.SyscallConn()
	if err != nil {
		return NilNetSocket, err
	}

	var broadcastErr error
	err = rc.Control(func(fd uintptr) {
		// enable boradcast option
		broadcastErr = syscall.SetsockoptInt(castFd(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
	})
	if err != nil {
		return NilNetSocket, err
	}

	if broadcastErr != nil {
		return NilNetSocket, broadcastErr
	}

	return NetSocket{
		socket: conn,
	}, nil
}

func (s NetSocket) IsValid() bool {
	return s.socket != nil
}

func (s *NetSocket) Close() error {
	return s.socket.Close()
}

func (s *NetSocket) WriteTo(addr netip.AddrPort, data []byte) error {
	var (
		sent = 0
		l    = len(data)
	)
	for sent < l {
		n, err := s.socket.WriteToUDPAddrPort(data, addr)
		if err != nil {
			return err
		}
		sent += n
	}
	return nil
}

func (s *NetSocket) ReadFrom(buf []byte) (n int, addr netip.AddrPort, err error) {
	return s.socket.ReadFromUDPAddrPort(buf)
}

/*
unsigned char aBuffer[NET_MAX_PACKETSIZE];

	dbg_assert(DataSize <= NET_MAX_PAYLOAD, "packet data size too high");
	dbg_assert((Token&~NET_TOKEN_MASK) == 0, "token out of range");
	dbg_assert((ResponseToken&~NET_TOKEN_MASK) == 0, "resp token out of range");

	int i = 0;
	aBuffer[i++] = ((NET_PACKETFLAG_CONNLESS<<2)&0xfc) | (NET_PACKETVERSION&0x03); // connless flag and version
	aBuffer[i++] = (Token>>24)&0xff; // token
	aBuffer[i++] = (Token>>16)&0xff;
	aBuffer[i++] = (Token>>8)&0xff;
	aBuffer[i++] = (Token)&0xff;
	aBuffer[i++] = (ResponseToken>>24)&0xff; // response token
	aBuffer[i++] = (ResponseToken>>16)&0xff;
	aBuffer[i++] = (ResponseToken>>8)&0xff;
	aBuffer[i++] = (ResponseToken)&0xff;

	dbg_assert(i == NET_PACKETHEADERSIZE_CONNLESS, "inconsistency");

	mem_copy(&aBuffer[i], pData, DataSize);
	net_udp_send(m_Socket, pAddr, aBuffer, i+DataSize);
*/
