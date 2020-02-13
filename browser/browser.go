package browser

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/jxsl13/twapi/compression"
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

	maxBufferSize             = 1500
	maxChunks                 = 16
	maxServersPerMasterServer = 75

	minTimeout = 60 * time.Millisecond
)

var (
	// Logger creates the logging output. The user may define an own logger, like a file or other.
	// the default value is the same as the standard logger from the "log" package
	Logger = log.New(os.Stderr, "", log.LstdFlags)

	// TimeoutMasterServers is used by ServerInfos as a value that drops few packets
	TimeoutMasterServers = 5 * time.Second

	// TimeoutServers is also used by ServerInfos as a alue that drops few packets
	TimeoutServers = TokenExpirationDuration

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

	// ErrTimeout is used in Retry functions that support a timeout parameter
	ErrTimeout = errors.New("timeout")

	// ErrInvalidWrite is returned if writing to an io.Writer failed
	ErrInvalidWrite = errors.New("invalid write")

	// ErrRequestResponseMismatch is returned by functions that request and receive data, but the received data does not match the requested data.
	ErrRequestResponseMismatch = errors.New("request response mismatch")

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

	masterServerHostnameAddresses = []string{"master1.teeworlds.com:8283", "master2.teeworlds.com:8283", "master3.teeworlds.com:8283", "master4.teeworlds.com:8283"}
	masterServerAddresses         = []*net.UDPAddr{}
)

// init initializes a package on import
func init() {
	masterServerAddresses = make([]*net.UDPAddr, 0, len(masterServerHostnameAddresses))

	for _, ms := range masterServerHostnameAddresses {
		srv, err := net.ResolveUDPAddr("udp", ms)
		if err != nil {
			Logger.Printf("Failed to resolve: %s\n", ms)
		} else {
			Logger.Printf("Resolved masterserver: %s -> %s\n", ms, srv.String())
			masterServerAddresses = append(masterServerAddresses, srv)
		}
	}
	if len(masterServerAddresses) == 0 {
		Logger.Fatalln("Could not resolve any masterservers.... terminating.")
	}
	return
}

// ReadWriteDeadliner narrows the used uparations of the passed type.
// in order to have a wider range of types that can satisfy this interface.
type ReadWriteDeadliner interface {
	io.ReadWriter

	// SetDeadline sets the read and write deadlines associated
	// with the connection. It is equivalent to calling both
	// SetReadDeadline and SetWriteDeadline.
	//
	// A deadline is an absolute time after which I/O operations
	// fail with a timeout (see type Error) instead of
	// blocking. The deadline applies to all future and pending
	// I/O, not just the immediately following call to Read or
	// Write. After a deadline has been exceeded, the connection
	// can be refreshed by setting a deadline in the future.
	//
	// An idle timeout can be implemented by repeatedly extending
	// the deadline after successful Read or Write calls.
	//
	// A zero value for t means I/O operations will not time out.
	//
	// Note that if a TCP connection has keep-alive turned on,
	// which is the default unless overridden by Dialer.KeepAlive
	// or ListenConfig.KeepAlive, then a keep-alive failure may
	// also return a timeout error. On Unix systems a keep-alive
	// failure on I/O can be detected using
	// errors.Is(err, syscall.ETIMEDOUT).
	SetDeadline(t time.Time) error

	// SetReadDeadline sets the deadline for future Read calls
	// and any currently-blocked Read call.
	// A zero value for t means Read will not time out.
	SetReadDeadline(t time.Time) error

	// SetWriteDeadline sets the deadline for future Write calls
	// and any currently-blocked Write call.
	// Even if write times out, it may return n > 0, indicating that
	// some of the data was successfully written.
	// A zero value for t means Write will not time out.
	SetWriteDeadline(t time.Time) error
}

// TokenRequestPacket can be sent to request a new token from the
type TokenRequestPacket []byte

// ServerListRequestPacket is used to request the server list from the masterserver
type ServerListRequestPacket []byte

// ServerCountRequestPacket is used to request the number of currently registered servers ad the masterserver
type ServerCountRequestPacket []byte

// ServerInfoRequestPacket is used to request the player and server information from a gameserver
type ServerInfoRequestPacket []byte

// ServerList is the result type of a serer list request
type ServerList []*net.UDPAddr

// ServerInfo contains the server's general information
type ServerInfo struct {
	Address     string       `json:"address"`
	Version     string       `json:"version"`
	Name        string       `json:"name"`
	Hostname    string       `json:"hostname,omitempty"`
	Map         string       `json:"map"`
	GameType    string       `json:"gametype"`
	ServerFlags int          `json:"server_flags"`
	SkillLevel  int          `json:"skill_level"`
	NumPlayers  int          `json:"num_players"`
	MaxPlayers  int          `json:"max_players"`
	NumClients  int          `json:"num_clients"`
	MaxClients  int          `json:"max_clients"`
	Players     []PlayerInfo `json:"players"`
}

// fix synchronizes the length of playerInfo with its struct field
func (s *ServerInfo) fix() {
	s.NumClients = len(s.Players)
}

// Equal compares two instances of ServerInfo and returns true if they are equal
func (s *ServerInfo) Equal(other ServerInfo) bool {
	s.fix()
	other.fix()
	equalData := s.Address == other.Address && s.Version == other.Version && s.Name == other.Name && s.Hostname == other.Hostname && s.Map == other.Map && s.GameType == other.GameType && s.ServerFlags == other.ServerFlags && s.SkillLevel == other.SkillLevel && s.NumPlayers == other.NumPlayers && s.MaxPlayers == other.MaxPlayers && s.NumClients == other.NumClients && s.MaxClients == other.MaxClients

	// equal Players
	if len(s.Players) != len(other.Players) {
		return false
	}
	for idx, p := range s.Players {
		if !p.Equal(other.Players[idx]) {
			return false
		}
	}
	return equalData
}

func (s *ServerInfo) String() string {
	s.fix()
	b, _ := json.Marshal(s)
	return string(b)
}

// MarshalBinary returns a binary representation of the ServerInfo
func (s *ServerInfo) MarshalBinary() (data []byte, err error) {
	s.fix()
	data = make([]byte, 0, maxBufferSize)

	data = append(data, []byte(s.Version)...)
	data = append(data, delimiter...)

	data = append(data, []byte(s.Name)...)
	data = append(data, delimiter...)

	data = append(data, []byte(s.Hostname)...)
	data = append(data, delimiter...)

	data = append(data, []byte(s.Map)...)
	data = append(data, delimiter...)

	data = append(data, []byte(s.GameType)...)
	data = append(data, delimiter...)

	data = append(data, byte(s.ServerFlags))
	data = append(data, byte(s.SkillLevel))

	var v compression.VarInt

	v.Pack(s.NumPlayers)
	v.Pack(s.MaxPlayers)
	v.Pack(len(s.Players)) // s.NumClients
	v.Pack(s.MaxClients)

	data = append(data, v.Bytes()...)
	v.Clear()

	for _, player := range s.Players {
		playerData, _ := player.marshalBinary()
		data = append(data, playerData...)
	}

	return
}

// UnmarshalBinary creates a serverinfo from binary data
func (s *ServerInfo) UnmarshalBinary(data []byte) (err error) {

	slots := bytes.SplitN(data, delimiter[:], 6) // create 6 slots
	if len(slots) != 6 {
		return fmt.Errorf("expected slots: 6 got: %d", len(slots))
	}

	s.Version = string(slots[0])
	s.Name = string(slots[1])
	s.Hostname = string(slots[2])
	s.Map = string(slots[3])
	s.GameType = string(slots[4])

	data = slots[5] // get next raw data chunk

	s.ServerFlags = int(data[0])
	s.SkillLevel = int(data[1])

	data = data[2:] // skip first two already evaluated bytes
	v := compression.NewVarIntFrom(data)
	s.NumPlayers, err = v.Unpack()
	if err != nil {
		return
	}
	s.MaxPlayers, err = v.Unpack()
	if err != nil {
		return
	}
	s.NumClients, err = v.Unpack()
	if err != nil {
		return
	}
	s.MaxClients, err = v.Unpack()
	if err != nil {
		return
	}

	// preallocate space for player pointers
	s.Players = make([]PlayerInfo, 0, s.NumClients)

	data = v.Bytes() // return the not yet used remaining data

	for i := 0; i < s.NumClients; i++ {
		player := PlayerInfo{}

		slots := bytes.SplitN(v.Bytes(), delimiter, 3) // create 3 slots
		if len(slots) != 3 {
			return fmt.Errorf("expected slots: 3 got: %d", len(slots))
		}

		player.Name = string(slots[0])
		player.Clan = string(slots[1])

		v = compression.NewVarIntFrom(slots[2])
		player.Country, err = v.Unpack()
		if err != nil {
			return
		}
		player.Score, err = v.Unpack()
		if err != nil {
			return
		}
		player.Type, err = v.Unpack()
		if err != nil {
			return
		}

		s.Players = append(s.Players, player)
	}
	return
}

// PlayerInfo contains a players externally visible information
type PlayerInfo struct {
	Name    string `json:"name"`
	Clan    string `json:"clan"`
	Type    int    `json:"type"`
	Country int    `json:"country"`
	Score   int    `json:"score"`
}

// Equal compares two instances for equality.
func (p *PlayerInfo) Equal(other PlayerInfo) bool {
	return p.Name == other.Name && p.Clan == other.Clan && p.Type == other.Type && p.Country == other.Country && p.Score == other.Score

}

func (p *PlayerInfo) String() string {
	b, _ := json.Marshal(p)
	return string(b)
}

// marshalBinary returns a binary representation of the PlayerInfo
// no delimiter is appended at the end of the byte slice
func (p *PlayerInfo) marshalBinary() (data []byte, err error) {

	data = make([]byte, 0, 2*len(delimiter)+len(p.Name)+len(p.Clan)+3*5)

	data = append(data, []byte(p.Name)...)
	data = append(data, delimiter...)

	data = append(data, []byte(p.Clan)...)
	data = append(data, delimiter...)

	var v compression.VarInt
	v.Pack(p.Country)
	v.Pack(p.Score)
	v.Pack(p.Type)

	data = append(data, v.Bytes()...)
	return
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
