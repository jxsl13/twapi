package browser

import (
	"bytes"
	"math/rand"
	"net"
	"time"

	"github.com/jxsl13/twapi/compression"
)

// NewTokenRequestPacket generates a new token request packet that can be
// used to request fo a new server token
func NewTokenRequestPacket() TokenRequestPacket {
	seedSource := rand.NewSource(time.Now().UnixNano())
	randomNumberGenerator := rand.New(seedSource)

	clientToken := int(randomNumberGenerator.Int31())
	serverToken := -1

	header := packTokenRequest(clientToken, serverToken)
	return TokenRequestPacket(header)
}

// NewServerListRequestPacket creates a new server list request packet
// Returns an ErrTokenExpired if the token passed alreay expired.
func NewServerListRequestPacket(t Token) (ServerListRequestPacket, error) {
	if t.Expired() {
		return ServerListRequestPacket{}, ErrTokenExpired
	}

	payload := make([]byte, 0, len(requestServerListRaw)+len(t.Payload))
	payload = append(payload, t.Payload...)
	payload = append(payload, requestServerListRaw...)
	return ServerListRequestPacket(payload), nil
}

// NewServerCountRequestPacket creates a new packet that can be used to request the number of currently registered
// game servers at the master servers
// Returns ErrTokenExpired if the passed token already expired
func NewServerCountRequestPacket(t Token) (ServerCountRequestPacket, error) {
	if t.Expired() {
		return ServerCountRequestPacket{}, ErrTokenExpired
	}

	payload := make([]byte, 0, len(requestServerCountRaw)+len(t.Payload))
	payload = append(payload, t.Payload...)
	payload = append(payload, requestServerCountRaw...)

	return ServerCountRequestPacket(payload), nil
}

// NewServerInfoRequestPacket creates a new request packet
// that can b eused to request the server info of a gameserver
func NewServerInfoRequestPacket(t Token) (ServerInfoRequestPacket, error) {
	if t.Expired() {
		return ServerInfoRequestPacket{}, ErrTokenExpired
	}

	payload := make([]byte, 0, len(requestInfoRaw)+len(t.Payload))
	payload = append(payload, t.Payload...)
	payload = append(payload, requestInfoRaw...)

	return ServerInfoRequestPacket(payload), nil
}

// ParseToken creates a new token from a response message that was sent by a server that was
// requested with a TokenRequestPacket
// Returns:
//		data without the token header
//		A token that is used for every continuous request, until the token expires and needs to be renewed
// 		an ErrInvalidResponseMessage if the serverResponse is too short.
// Info: If the serverResponse is incorrect, but has the correct length, the resulting token might contain invalid data.
// This function should be immediatly called after receiving the Token Response message from the server.
func ParseToken(serverResponse []byte) (Token, error) {
	tokenClient, tokenServer, err := unpackTokenResponse(serverResponse)
	if err != nil {
		return Token{}, err
	}

	header := packToken(tokenClient, tokenServer)

	return Token{header, time.Now().Add(TokenExpirationDuration - 1*time.Second), tokenClient, tokenServer}, nil
}

// ParseServerList parses the response server list
func ParseServerList(serverResponse []byte) (ServerList, error) {
	if len(serverResponse) < tokenPrefixSize+len(sendServerListRaw) {
		return nil, ErrInvalidResponseMessage
	}

	//newTokenFromFollowUpRequest(serverResponse[:tokenPrefixSize])

	responseHeaderRaw := serverResponse[tokenPrefixSize : tokenPrefixSize+len(sendServerListRaw)]

	if !bytes.Equal(responseHeaderRaw, sendServerListRaw) {
		return nil, ErrUnexpectedResponseHeader
	}

	data := serverResponse[tokenPrefixSize+len(sendServerListRaw):]

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
		ip := []byte{}

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
func ParseServerCount(serverResponse []byte) (int, error) {
	if len(serverResponse) < tokenPrefixSize+len(sendServerListRaw) {
		return 0, ErrInvalidResponseMessage
	}

	responseHeaderRaw := serverResponse[tokenPrefixSize : tokenPrefixSize+len(sendServerListRaw)]

	if !bytes.Equal(responseHeaderRaw, sendServerCountRaw) {
		return 0, ErrUnexpectedResponseHeader
	}

	data := serverResponse[tokenPrefixSize+len(sendServerListRaw):]

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
func ParseServerInfo(serverResponse []byte, address string) (info ServerInfo, err error) {
	if len(serverResponse) < tokenPrefixSize+len(sendInfoRaw) {
		err = ErrInvalidResponseMessage
		return
	}

	responseHeaderRaw := serverResponse[tokenPrefixSize : tokenPrefixSize+len(sendInfoRaw)]

	if !bytes.Equal(responseHeaderRaw, sendInfoRaw) {
		err = ErrUnexpectedResponseHeader
		return
	}

	data := serverResponse[tokenPrefixSize+len(sendInfoRaw):]

	slots := bytes.SplitN(data, delimiter[:], 6) // create 6 slots
	if len(slots) != 6 {
		err = ErrInvalidResponseMessage
		return
	}

	info.Address = address
	info.Version = string(slots[0])
	info.Name = string(slots[1])
	info.Hostname = string(slots[2])
	info.Map = string(slots[3])
	info.GameType = string(slots[4])

	data = slots[5] // get next raw data chunk

	info.ServerFlags = int(data[0])
	info.SkillLevel = int(data[1])

	data = data[2:] // skip first two already evaluated bytes
	v := compression.NewVarIntFrom(data)
	info.NumPlayers, err = v.Unpack()
	if err != nil {
		err = ErrInvalidResponseMessage
		return
	}
	info.MaxPlayers, err = v.Unpack()
	if err != nil {
		err = ErrInvalidResponseMessage
		return
	}
	info.NumClients, err = v.Unpack()
	if err != nil {
		err = ErrInvalidResponseMessage
		return
	}
	info.MaxClients, err = v.Unpack()
	if err != nil {
		err = ErrInvalidResponseMessage
		return
	}

	// preallocate space for player pointers
	info.Players = make([]PlayerInfo, 0, info.NumClients)

	data = v.Data() // return the not yet used remaining data

	for i := 0; i < info.NumClients; i++ {
		player := PlayerInfo{}

		slots := bytes.SplitN(v.Data(), []byte("\x00"), 3) // create 3 slots
		if len(slots) != 3 {
			err = ErrInvalidResponseMessage
			return
		}

		player.Name = string(slots[0])
		player.Clan = string(slots[1])

		v = compression.NewVarIntFrom(slots[2])
		player.Country, err = v.Unpack()
		if err != nil {
			err = ErrInvalidResponseMessage
			return
		}
		player.Score, err = v.Unpack()
		if err != nil {
			err = ErrInvalidResponseMessage
			return
		}
		player.Type, err = v.Unpack()
		if err != nil {
			err = ErrInvalidResponseMessage
			return
		}

		info.Players = append(info.Players, player)
	}

	return info, nil
}

// packs header
func packTokenRequest(tokenClient, tokenServer int) []byte {
	const netPacketFlagControl = 1
	const netControlMessageToken = 5
	const netTokenRequestDataSize = 512

	const size = 4 + 3 + netTokenRequestDataSize

	a := [size]byte{}
	b := a[:]

	// Header
	b[0] = (netPacketFlagControl << 2) & 0b11111100
	b[3] = byte(tokenServer >> 24)
	b[4] = byte(tokenServer >> 16)
	b[5] = byte(tokenServer >> 8)
	b[6] = byte(tokenServer)
	// Data
	b[7] = netControlMessageToken
	b[8] = byte(tokenClient >> 24)
	b[9] = byte(tokenClient >> 16)
	b[10] = byte(tokenClient >> 8)
	b[11] = byte(tokenClient)
	return b
}

// retrieve token from specific "token response" message.
// that message is the explicit answer to the token request
func unpackTokenResponse(message []byte) (tokenClient, tokenServer int, err error) {
	if len(message) < tokenResponseSize {
		err = ErrInvalidHeaderLength
		return
	}

	tokenClient = (int(message[3]) << 24) + (int(message[4]) << 16) + (int(message[5]) << 8) + int(message[6])
	tokenServer = (int(message[8]) << 24) + (int(message[9]) << 16) + (int(message[10]) << 8) + int(message[11])
	return
}

func packToken(tokenClient, tokenServer int) (header []byte) {
	const netPacketFlagConnless = 8
	const netPacketVersion = 1

	a := [tokenPrefixSize]byte{}
	header = a[:]

	// Header
	header[0] = ((netPacketFlagConnless << 2) & 0b11111100) | (netPacketVersion & 0b00000011)
	header[1] = byte(tokenServer >> 24)
	header[2] = byte(tokenServer >> 16)
	header[3] = byte(tokenServer >> 8)
	header[4] = byte(tokenServer)
	// ResponseToken
	header[5] = byte(tokenClient >> 24)
	header[6] = byte(tokenClient >> 16)
	header[7] = byte(tokenClient >> 8)
	header[8] = byte(tokenClient)

	return
}
