package browser

import (
	"bytes"
	"net"
)

// parseServerList parses the response server list
func parseServerList(serverListPayoad []byte) ([]*net.UDPAddr, error) {
	data := serverListPayoad
	/*
		each server information contains of 18 bytes
		first 16 bytes define the IP
		the last 2 bytes define the port

		if the first 12 bytes match the defined pefix, the IP is parsed as IPv4
		and if it does not match, the IP is parsed as IPv6
	*/
	numServers := len(data) / 18 // 18 bytes, 16 for IPv4/IPv6 and 2 bytes for the port
	serverList := make([]*net.UDPAddr, 0, numServers)

	// fixed size array
	ipv4Prefix := [12]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF}

	for idx := 0; idx < numServers; idx++ {
		var ip []byte

		// compare byte slices
		if bytes.Equal(ipv4Prefix[:], data[idx*18:idx*18+12]) {
			// IPv4 has a prefix
			ip = data[idx*18+12 : idx*18+16]
		} else {
			// full IP is the IPv6 otherwise
			ip = data[idx*18 : idx*18+16]
		}

		serverList = append(serverList, &net.UDPAddr{
			IP:   ip,
			Port: (int(data[idx*18+16]) << 8) + int(data[idx*18+17]),
		})
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
func parseServerInfo(serverInfoPayload []byte, address string) (*ServerInfo, error) {
	data := serverInfoPayload

	info := &ServerInfo{
		Address: address,
	}
	err := info.UnmarshalBinary(data)
	if err != nil {
		return nil, err
	}
	return info, nil
}
