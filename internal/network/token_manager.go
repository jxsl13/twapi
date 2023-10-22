package network

import (
	crand "crypto/rand"
	"math"
	"math/big"
	"math/rand"
	"time"

	"github.com/jxsl13/twapi/protocol"
)

type TokenManager struct {
	seedInterval time.Duration
	nextSeedTime time.Time

	prevSeed int64
	seed     int64

	prevGlobalToken protocol.Token
	globalToken     protocol.Token
}

func NewTokenManager() *TokenManager {
	tm := TokenManager{
		seedInterval: protocol.NetSeedTime,
	}

	return &tm
}

func (t *TokenManager) GenerateToken(addr NetAddr) protocol.Token {
	return GenerateToken(addr, t.seed)
}

func (t *TokenManager) GenerateSeed() {
	t.prevSeed = t.seed

	i, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		// fallback to pseudo random
		t.seed = rand.New(rand.NewSource(time.Now().Unix())).Int63()
	} else {
		t.seed = i.Int64()
	}

	t.prevGlobalToken = t.globalToken
	t.globalToken = t.GenerateToken(NilNetAddr)

	t.nextSeedTime = time.Now().Add(t.seedInterval)
}

func (t *TokenManager) Update() {
	if time.Now().After(t.nextSeedTime) {
		t.GenerateSeed()
	}
}

func (t *TokenManager) CheckToken(addr NetAddr, token protocol.Token, respToken protocol.Token) (ok bool, isBroadcast bool) {

	expectedToken := GenerateToken(addr, t.seed)
	if expectedToken == token {
		return true, false
	}
	prevExpectedToken := GenerateToken(addr, t.prevSeed)
	if prevExpectedToken == token {
		// no need to notify the peer, just a one time thing
		return true, false
	} else if token == t.globalToken {
		return true, true
	} else if token == t.prevGlobalToken {
		// no need to notify the peer, just a broadcast token response
		return true, true
	}

	return false, false
}

func (t *TokenManager) ProcessMessage(addr NetAddr, packet *PacketConstruct) int {
	// TODO: need the ability to send udp packets in here
	// missing CNetBase *pNetBase struct field
	return 0

}
