package protocol

import "time"

const (
	// NetTokenMax is the biggest possible but invalid token
	// Only tokens below this value are valid
	NetTokenMax Token = 0xffffffff

	// NetTokenNone is the token that is used when you do not have
	// yet received a token from a server. It is the same as the invalid token.
	// It has a single purpose: indicating that your client needs a new token.
	NetTokenNone Token = NetTokenMax

	// NetTokenMask is the mask to keep tokens between 0 and  NetTokenMax (including)
	// In the edge case that the resulting token is NetTokenMax, it should be either
	// deterministically changed to be reproducable.
	NetTokenMask Token = NetTokenMax

	// NetSeedTime is the iduration between the regeneration of a new seed.
	NetSeedTime = 16 * time.Second

	// NetTokenRequestDataSize is the exact size (with 0 padding)
	// that a token request has to occupy.
	NetTokenRequestDataSize = 512
)

// Token represents a security token that is periodocally rotated (NetSeedTime)
// and has to be sent with every other package that is being sent to the Teeworlds server.
type Token uint32
