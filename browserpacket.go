package main

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"
)

const (
	// Used for the masterserver
	requestServerList  = "\xff\xff\xff\xffreq2"
	sendServerList     = "\xff\xff\xff\xfflis2"
	requestServerCount = "\xff\xff\xff\xffcou2"
	sendServerCount    = "\xff\xff\xff\xffsiz2"

	// Used for the gameserver
	requestInfo = "\xff\xff\xff\xffgie3\x00" // need explicitly the trailing \x00
	sendInfo    = "\xff\xff\xff\xffinf3\x00"

	tokenResponseSize = 12
	tokenRequestSize  = 9
)

var (
	// ErrTokenExpired is returned when a request packet is being constructed with an expired token
	ErrTokenExpired = errors.New("token expired")

	// ErrInvalidResponseMessage is returned when a passed response message does not contain the expected data.
	ErrInvalidResponseMessage = errors.New("invalid response message")

	// TokenExpirationDuration sets the protocol expiration time of a token
	// This variable can be changed
	TokenExpirationDuration = time.Second * 90

	requestServerListRaw  = []byte(string(requestServerList))
	sendServerListRaw     = []byte(string(sendServerList))
	requestServerCountRaw = []byte(string(requestServerCount))
	sendServerCountRaw    = []byte(string(sendServerCount))
	requestInfoRaw        = []byte(string(requestInfo))
	sendInfoRaw           = []byte(string(sendInfo))
)

// TokenRequestPacket can be sent to request a new token from the
type TokenRequestPacket []byte

// ServerListRequestPacket is used to request the server list from the masterserver
type ServerListRequestPacket []byte

// ServerCountRequestPacket is used to request the number of currently registered servers ad the masterserver
type ServerCountRequestPacket []byte

// ServerInfoRequestPacket is used to request the player and server information from a gameserver
type ServerInfoRequestPacket []byte

// ServerList is the result type of a serer list request
type ServerList []net.UDPAddr

// type ServerInfo struct {
// 	//...
// 	Players []PlayerInfo
// }
// type PlayerInfo struct {
// 	//...
// }

// Token is used to request information from either master of game servers.
// The token needs to be renewed via NewTokenRequestPacket()
// followed by parsing the server's response with NewToken(responseMessage []byte) (Token, error)
type Token struct {
	Payload   []byte //len should be 12 at most
	expiresAt time.Time
	client    int
	server    int
}

// Expired returns true if the token already expired and needs to be renewed
func (ts *Token) Expired() bool {
	return ts.expiresAt.Before(time.Now())
}

// NewTokenRequestPacket generates a new token request packet that can be
// used to request fo a new server token
func NewTokenRequestPacket() TokenRequestPacket {
	seedSource := rand.NewSource(time.Now().UnixNano())
	randomNumberGenerator := rand.New(seedSource)

	clientToken := int(randomNumberGenerator.Int31())
	clientToken = 66666
	serverToken := -1

	header := packTokenRequest(clientToken, serverToken)
	return TokenRequestPacket(header)
}

// NewToken creates a new token from a response message that was sent by a server that was
// requested with a TokenRequestPacket
// Returns:
//		data without the token header
//		A token that is used for every continuous request, until the token expires and needs to be renewed
// 		an ErrInvalidResponseMessage if the serverResponse is too short.
// Info: If the serverResponse is incorrect, but has the correct length, the resulting token might contain invalid data.
// This function should be immediatly called after receiving the Token Response message from the server.
func NewToken(serverResponse []byte) (Token, error) {
	tokenClient, tokenServer, err := unpackTokenResponse(serverResponse)
	if err != nil {
		return Token{}, ErrInvalidResponseMessage
	}

	header := packToken(tokenClient, tokenServer)

	return Token{header, time.Now().Add(TokenExpirationDuration - 1*time.Second), tokenClient, tokenServer}, nil
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

// NewServerList parses the response server list
func NewServerList(serverResponse []byte) (ServerList, Token, error) {
	if len(serverResponse) < tokenResponseSize+len(sendServerListRaw) {
		return nil, Token{}, ErrInvalidResponseMessage
	}

	token, err := NewToken(serverResponse[:tokenResponseSize])
	if err != nil {
		return nil, Token{}, err
	}

	responseHeader := string(serverResponse[tokenResponseSize : tokenResponseSize+len(sendServerListRaw)])

	if strings.Compare(responseHeader, "\xff\xff\xff\xfflis2") != 0 {
		return nil, Token{}, ErrInvalidResponseMessage
	}

	data := serverResponse[tokenResponseSize+len(sendServerListRaw):]

	/*
		each server information contains of 18 bytes
		first 16 bytes define the IP
		the last 2 bytes define the port

		if the first 12 bytes match the defined pefix, the IP is parsed as IPv4
		and if it does not match, the IP is parsed as IPv6
	*/
	numServers := len(data) / 18 // 18 byte, 16 for IPv4/IPv6 and 2 bytes for the port
	serverList := make([]net.UDPAddr, 0, numServers)

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

		serverList = append(serverList, net.UDPAddr{
			IP:   ip,
			Port: (int(data[idx*18+16]) << 8) + int(data[idx*18+17]),
		})
	}

	return serverList, token, nil

}

// func NewServerCount(serverResponse []byte) (int, Token, error) {

// }

// func NewServerInfo(serverResponse []byte) (ServerInfo, Token, error) {

// }

// packs header
func packTokenRequest(tokenClient, tokenServer int) []byte {
	const netPacketFlagControl = 1
	const netControlMessageToken = 5
	const netTokenRequestDataSize = 512

	const size = 4 + 3 + netTokenRequestDataSize
	b := make([]byte, size)

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

// unpacks header
func unpackTokenResponse(message []byte) (tokenClient, tokenServer int, err error) {
	if len(message) < tokenResponseSize {
		err = fmt.Errorf("control message is too small, %d byte, required %d byte", len(message), tokenResponseSize)
		return
	}
	tokenClient = (int(message[3]) << 24) + (int(message[4]) << 16) + (int(message[5]) << 8) + int(message[6])
	tokenServer = (int(message[8]) << 24) + (int(message[9]) << 16) + (int(message[10]) << 8) + int(message[11])
	return
}

func packToken(tokenClient, tokenServer int) (header []byte) {
	const netPacketFlagConnless = 8
	const netPacketVersion = 1

	header = make([]byte, tokenRequestSize)

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
