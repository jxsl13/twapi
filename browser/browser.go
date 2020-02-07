package browser

import (
	"bytes"
	"errors"
	"fmt"
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
