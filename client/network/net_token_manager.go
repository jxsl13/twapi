package network

import (
	"math/rand"
	"time"
	"binary"

	"inet.af/netaddr"
)

type NetTokenManager struct {
	NetBase *NetBase

	Seed     int64
	PrevSeed int64

	GlobalToken     Token
	PrevGlobalToken Token

	SeedTime     int
	NextSeedTime time.Time
}

func (ntm *NetTokenManager) Init(pNetBase *NetBase, OptionalSeedTime ...int) {
	SeedTime := 0
	if len(OptionalSeedTime) == 0 {
		// default const value
		SeedTime = NetSeedTime
	} else {
		SeedTime = OptionalSeedTime[0]
	}

	ntm.NetBase = pNetBase
	ntm.SeedTime = SeedTime

}

func (ntm *NetTokenManager) Update() {

}

var (
	NullAddr = netaddr.IPPort{}
)

func (ntm *NetTokenManager) GenerateSeed() {

	ntm.PrevSeed = ntm.Seed
	ntm.Seed = rand.Int63()

	ntm.PrevGlobalToken = ntm.GlobalToken
	ntm.GlobalToken = GenerateToken(&NullAddr)

	ntm.NextSeedTime = time.Now().Add(time.Second * time.Duration(ntm.SeedTime))
}

func (ntm *NetTokenManager) ProcessMessage(pAddr netaddr.IPPort, pPacket *NetPacketConstruct) int {
	return 0
}
func (ntm *NetTokenManager) CheckToken(pAddr netaddr.IPPort, token Token, ResponseToken Token, BroadcastResponse *bool) bool {
	return false
}
func (ntm *NetTokenManager) GenerateToken(pAddr netaddr.IPPort) Token {
	return 0
}

func (ntm *NetTokenManager) GenerateToken(pAddr netaddr.IPPort) Token {
	return GenerateToken(pAddr, ntm.Seed)
}

func GenerateToken(pAddr netaddr.IPPort, Seed int64) Token {
	Addr := pAddr

	aBuf := make([]byte, 0)
	var Result uint

	aBuf = append(aBuf, Addr.UDPAddr().IP...)
	aBuf = append(aBuf, binary.LittleEndian.PutInt64(bs, Seed))

	mem_copy(aBuf + sizeof(NETADDR), &Seed, sizeof(int64));

	Result = Hash(aBuf, sizeof(aBuf)) & NET_TOKEN_MASK;
	if(Result == NET_TOKEN_NONE)
		Result--;

	return Result;
}
