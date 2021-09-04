package browser

import (
	"crypto/rand"
	"fmt"
	"time"
)

// NewTokenRequestPacket generates a new token request packet that can be
// used to request for a new server token
func NewTokenRequestPacket() []byte {

	n, _ := rand.Read(nil)

	t := Token{
		ServerToken: -1,
		ClientToken: n,
	}

	header, _ := t.MarshalBinary()

	const netPacketFlagControl = 1
	const netControlMessageToken = 5
	const netTokenRequestDataSize = 512

	const size = 4 + 3 + netTokenRequestDataSize // 519 = token request expected size

	buffer := [size]byte{} // stack allocation
	payload := buffer[:]

	// Header
	payload[0] = (netPacketFlagControl << 2) & 0b11111100

	payload[3] = header[1]
	payload[4] = header[2]
	payload[5] = header[3]
	payload[6] = header[4]
	// Data
	payload[7] = netControlMessageToken
	payload[8] = header[5]
	payload[9] = header[6]
	payload[10] = header[7]
	payload[11] = header[8]
	return payload
}

// Token is used to request information from either master of game servers.
// The token needs to be renewed via NewTokenRequestPacket()
// followed by parsing the server's response with NewToken(responseMessage []byte) (Token, error)
type Token struct {
	//Payload   []byte //len should be 12 at most
	ExpiresAt   time.Time
	ClientToken int
	ServerToken int
}

// Expired returns true if the token already expired and needs to be renewed
func (ts *Token) Expired() bool {
	if ts == nil {
		return true
	}
	return ts.ExpiresAt.Before(time.Now())
}

// Equal tests if two token contain the same payload
func (ts *Token) Equal(t Token) bool {
	return ts.ClientToken == t.ClientToken && ts.ServerToken == t.ServerToken
}

// String implements the Stringer interface and returns a stringrepresentation of the token
func (ts *Token) String() string {
	return fmt.Sprintf("token: ClientToken: %d server: %d expires: %s", ts.ClientToken, ts.ServerToken, ts.ExpiresAt.String())
}

// MarshalBinary does satisfy the token prefix for secondary requests,
// but not for the initial token request
func (ts *Token) MarshalBinary() ([]byte, error) {
	header := [tokenPrefixSize]byte{}

	// Header
	header[0] = ((netPacketFlagConnless << 2) & 0b11111100) | (netPacketVersion & 0b00000011)
	header[1] = byte(ts.ServerToken >> 24)
	header[2] = byte(ts.ServerToken >> 16)
	header[3] = byte(ts.ServerToken >> 8)
	header[4] = byte(ts.ServerToken)
	// ResponseToken
	header[5] = byte(ts.ClientToken >> 24)
	header[6] = byte(ts.ClientToken >> 16)
	header[7] = byte(ts.ClientToken >> 8)
	header[8] = byte(ts.ClientToken)

	return header[:], nil
}

func (ts *Token) UnmarshalBinary(data []byte) error {
	if len(data) < tokenResponseSize {
		return ErrInvalidHeaderLength
	}

	ts.ClientToken = (int(data[3]) << 24) + (int(data[4]) << 16) + (int(data[5]) << 8) + int(data[6])
	ts.ServerToken = (int(data[8]) << 24) + (int(data[9]) << 16) + (int(data[10]) << 8) + int(data[11])

	// unmarshaling means that we received a new token from the server
	ts.ExpiresAt = time.Now().Add(TokenExpirationDuration - time.Second)
	return nil
}
