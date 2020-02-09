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

	minTimeout = 35 * time.Millisecond
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

// Valid returns true is the struct contains valid data
func (s *ServerInfo) Valid() bool {
	return s.Address != "" && s.Map != ""
}

func (s *ServerInfo) String() string {
	b, _ := json.Marshal(s)
	return string(b)
}

// PlayerInfo contains a players externally visible information
type PlayerInfo struct {
	Name    string `json:"name"`
	Clan    string `json:"clan"`
	Type    int    `json:"type"`
	Country int    `json:"country"`
	Score   int    `json:"score"`
}

func (p *PlayerInfo) String() string {
	b, _ := json.Marshal(p)
	return string(b)
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
