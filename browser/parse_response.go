package browser

import (
	"bytes"
	"fmt"
	"net/netip"
)

var (
	// fixed size array
	ipv4Prefix      = [12]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF}
	ipv4PrefixSlice = ipv4Prefix[:]
)

// parseServerList parses the response server list
func parseServerList(serverListPayoad []byte) ([]netip.AddrPort, error) {
	data := serverListPayoad
	/*
		each server information contains of 18 bytes
		first 16 bytes define the IP
		the last 2 bytes define the port

		if the first 12 bytes match the defined pefix, the IP is parsed as IPv4
		and if it does not match, the IP is parsed as IPv6
	*/
	numServers := len(data) / 18 // 18 bytes, 16 for IPv4/IPv6 and 2 bytes for the port
	serverList := make([]netip.AddrPort, 0, numServers)

	var (
		ok   bool
		addr netip.Addr
		port uint16
		ip   []byte

		ipStart, ipv4PrefixEnd, ipEnd int
		portHigh, portLow             int
	)

	for idx := 0; idx < numServers; idx++ {
		// calculate index values
		ipStart = idx * 18
		ipv4PrefixEnd = idx*18 + 12
		ipEnd = idx*18 + 16 // ipv6 has 16 bytes
		portHigh, portLow = ipEnd, idx*18+17

		// compare byte slices
		if bytes.Equal(ipv4PrefixSlice, data[ipStart:ipv4PrefixEnd]) {
			ipv4Start := ipv4PrefixEnd
			// IPv4 has a prefix
			ip = data[ipv4Start:ipEnd]
		} else {
			// full IP is the IPv6 otherwise
			ip = data[ipStart:ipEnd]
		}

		addr, ok = netip.AddrFromSlice(ip)
		if !ok {
			return nil, fmt.Errorf("invalid ip address bytes: %v", ip)
		}

		port = (uint16(data[portHigh]) << 8) + (uint16(data[portLow]))

		serverList = append(serverList, netip.AddrPortFrom(addr, port))
	}

	return serverList, nil
}

// ParseServerCount parses the response and returns the number of currently registered servers.
func parseServerCount(serverCountPayload []byte) (int, error) {
	data := serverCountPayload

	if len(data) > 4 {
		return 0, ErrInvalidResponseMessage
	}

	count := 0
	for idx, b := range data {
		count |= (int(b) << ((len(data) - 1) - idx))
	}

	return count, nil
}

// ParseServerInfo parses the serrver's server info response
func parseServerInfo(serverInfoPayload []byte, address string) (ServerInfo, error) {
	data := serverInfoPayload

	info := ServerInfo{
		Address: address,
	}
	err := info.UnmarshalBinary(data)
	if err != nil {
		return info, err
	}
	return info, nil
}
