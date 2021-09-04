package browser

import (
	"math"
	"net"
	"strconv"
	"strings"
)

func parseAddress(address string) (*net.UDPAddr, error) {
	parts := strings.Split(address, ":")
	if len(parts) != 2 {
		return nil, ErrInvalidAddress
	}
	ip := net.ParseIP(parts[0])
	if ip == nil {
		// address not an ip, try resolving hostname
		return net.ResolveUDPAddr("udp", address)
	}

	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, ErrInvalidPort
	}

	if port < 0 || math.MaxUint16 < port {
		return nil, ErrInvalidPort
	}

	addr := &net.UDPAddr{
		IP:   ip,
		Port: port,
	}
	return addr, nil
}
