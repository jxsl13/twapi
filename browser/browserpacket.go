package browser

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/jxsl13/twapi/compression"
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

	minHeaderLength = 8 // length of the shortest header
	maxHeaderLength = 9 // length of the longest header

	tokenResponseSize = 12 // size of the token that is sent after the TokenRequest
	tokenPrefixSize   = 9  // size of the token that is sent as prefix to every follow up message.

	minPrefixLength = tokenResponseSize
	maxPrefixLength = tokenPrefixSize + maxHeaderLength
)

var (
	// ErrTokenExpired is returned when a request packet is being constructed with an expired token
	ErrTokenExpired = errors.New("token expired")

	// ErrInvalidResponseMessage is returned when a passed response message does not contain the expected data.
	ErrInvalidResponseMessage = errors.New("invalid response message")

	// ErrInvalidHeaderLength is returned, if a too short byte slice is passed to some of the parsing methods
	ErrInvalidHeaderLength = errors.New("invalid header length")

	// ErrInvalidHeaderFlags is returned, if the first byte of a response message does not corespond to the expected flags.
	ErrInvalidHeaderFlags = errors.New("invalid header flags")

	// ErrUnexpectedResponseHeader is returned, if a message is passed to a parsing function, that expects a different response
	ErrUnexpectedResponseHeader = errors.New("unexpected response header")

	// TokenExpirationDuration sets the protocol expiration time of a token
	// This variable can be changed
	TokenExpirationDuration = time.Second * 16

	requestServerListRaw  = []byte(requestServerList)
	sendServerListRaw     = []byte(sendServerList)
	requestServerCountRaw = []byte(requestServerCount)
	sendServerCountRaw    = []byte(sendServerCount)
	requestInfoRaw        = []byte(requestInfo)
	sendInfoRaw           = []byte(sendInfo)
	delimiter             = []byte("\x00")
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

// ServerInfo contains the server's general information
type ServerInfo struct {
	Address     string
	Version     string
	Name        string
	Hostname    string
	Map         string
	GameType    string
	ServerFlags int
	SkillLevel  int
	NumPlayers  int
	MaxPlayers  int
	NumClients  int
	MaxClients  int
	Players     []PlayerInfo
	Date        time.Time
}

// Valid returns true is the struct contains valid data
func (s *ServerInfo) Valid() bool {
	return s.Address != "" && s.Map != ""
}

func (s *ServerInfo) String() string {
	base := fmt.Sprintf("\nHostname: %10s\nAddress: %20s\nVersion: '%10s'\nName: '%s'\nGameType: %s\nMap: %s\nServerFlags: %b\nSkilllevel: %d\n%d/%d Players \n%d/%d Clients\nDate: %s\n",
		s.Hostname,
		s.Address,
		s.Version,
		s.Name,
		s.GameType,
		s.Map,
		s.ServerFlags,
		s.SkillLevel,
		s.NumPlayers,
		s.MaxPlayers,
		s.NumClients,
		s.MaxClients,
		s.Date.Local().String())
	sb := strings.Builder{}
	sb.Grow(256 + s.NumClients*128)
	sb.WriteString(base)

	for _, p := range s.Players {
		sb.WriteString(p.String())
	}
	return sb.String()
}

// PlayerInfo contains a players externally visible information
type PlayerInfo struct {
	Name    string
	Clan    string
	Type    int
	Country int
	Score   int
}

func (p *PlayerInfo) String() string {
	return fmt.Sprintf("Name=%27s Clan=%13s Type=%1d Country=%3d Score=%6d\n", p.Name, p.Clan, p.Type, p.Country, p.Score)
}

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

// Equal tests if two token contain the same payload
func (ts *Token) Equal(t Token) bool {
	return bytes.Equal(ts.Payload, t.Payload)
}

// String implements the Stringer interface and returns a stringrepresentation of the token
func (ts *Token) String() string {
	return fmt.Sprintf("Token(%d): Client: %d Server: %d Expires: %s", len(ts.Payload), ts.client, ts.server, ts.expiresAt.String())
}

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

// MatchResponse matches a respnse to a specific string
// "", ErrInvalidResponseMessage -> if response message contains invalid data
// "", ErrInvalidHeaderLength -> if response message is too short
// "token" - token response
// "serverlist" - server list response
// "servercount" - server count response
// "serverinfo" - server info response
func MatchResponse(responseMessage []byte) (string, error) {
	if len(responseMessage) < minPrefixLength {
		return "", ErrInvalidHeaderLength
	}

	if len(responseMessage) == 12 {
		return "token", nil
	} else if bytes.Equal(sendServerListRaw, responseMessage[tokenPrefixSize:tokenPrefixSize+len(sendServerListRaw)]) {
		return "serverlist", nil
	} else if bytes.Equal(sendServerCountRaw, responseMessage[tokenPrefixSize:tokenPrefixSize+len(sendServerCountRaw)]) {
		return "servercount", nil
	} else if bytes.Equal(sendInfoRaw, responseMessage[tokenPrefixSize:tokenPrefixSize+len(sendInfoRaw)]) {
		return "serverinfo", nil
	}
	return "", ErrInvalidResponseMessage
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
func ParseServerInfo(serverResponse []byte, address net.Addr) (ServerInfo, error) {
	if len(serverResponse) < tokenPrefixSize+len(sendInfoRaw) {
		return ServerInfo{}, ErrInvalidResponseMessage
	}

	responseHeaderRaw := serverResponse[tokenPrefixSize : tokenPrefixSize+len(sendInfoRaw)]

	if !bytes.Equal(responseHeaderRaw, sendInfoRaw) {
		return ServerInfo{}, ErrUnexpectedResponseHeader
	}

	data := serverResponse[tokenPrefixSize+len(sendInfoRaw):]

	info := ServerInfo{}

	slots := bytes.SplitN(data, delimiter[:], 6) // create 6 slots

	info.Address = address.String()
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
	info.NumPlayers = v.Unpack()
	info.MaxPlayers = v.Unpack()
	info.NumClients = v.Unpack()
	info.MaxClients = v.Unpack()

	// preallocate space for player pointers
	info.Players = make([]PlayerInfo, 0, info.NumClients)

	data = v.Data() // return the not yet used remaining data

	for i := 0; i < info.NumClients; i++ {
		player := PlayerInfo{}

		slots := bytes.SplitN(v.Data(), []byte("\x00"), 3) // create 3 slots

		player.Name = string(slots[0])
		player.Clan = string(slots[1])

		v = NewVarIntFrom(slots[2])
		player.Country = v.Unpack()
		player.Score = v.Unpack()
		player.Type = v.Unpack()

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

// // Followup requests have a different token representation than the initial token request.
// func newTokenFromFollowUpRequest(serverResponse []byte) (Token, error) {
// 	tokenClient, tokenServer, err := unpackToken(serverResponse)

// 	if err != nil {
// 		return Token{}, err
// 	}

// 	header := packToken(tokenClient, tokenServer)

// 	return Token{header, time.Now().Add(TokenExpirationDuration - 1*time.Second), tokenClient, tokenServer}, nil
// }

// func unpackToken(message []byte) (tokenClient, tokenServer int, err error) {
// 	if len(message) < tokenPrefixSize {
// 		err = ErrInvalidHeaderLength
// 		return
// 	}

// 	const netPacketFlagConnless = 8
// 	const netPacketVersion = 1
// 	const headerFlags = ((netPacketFlagConnless << 2) & 0b11111100) | (netPacketVersion & 0b00000011)

// 	if message[0] != headerFlags {
// 		err = ErrInvalidHeaderFlags
// 		return
// 	}

// 	tokenClient = (int(message[1]) << 24) + (int(message[2]) << 16) + (int(message[3]) << 8) + int(message[4])
// 	tokenServer = (int(message[5]) << 24) + (int(message[6]) << 16) + (int(message[7]) << 8) + int(message[8])
// 	return
// }

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
