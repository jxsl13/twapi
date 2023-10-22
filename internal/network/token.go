package network

import (
	"crypto/md5"
	"encoding/binary"

	"github.com/jxsl13/twapi/protocol"
)

func GenerateToken(addr NetAddr, seed int64) protocol.Token {
	nilAddr := NilNetAddr

	if addr.IsBroadcast() {
		return GenerateToken(nilAddr, seed)
	}

	nilAddr.SetAddr(addr.Addr())
	// ignores port & broadcast

	buf, _ := nilAddr.MarshalBinary()
	buf = binary.LittleEndian.AppendUint64(buf, uint64(seed))

	sum := md5.Sum(buf)
	token := protocol.Token(binary.BigEndian.Uint32(sum[:4])) & protocol.NetTokenMask
	if token == protocol.NetTokenMax {
		token--
	}

	return token
}
