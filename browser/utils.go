package browser

import (
	"net"
)

func resolveUDPAddr(address string) (*net.UDPAddr, error) {
	return net.ResolveUDPAddr("udp", address)
}
