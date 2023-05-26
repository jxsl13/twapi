package compression

import "github.com/jxsl13/twapi/protocol"

type MsgPacker struct {
	Packer
}

// NewMsgPacker creates a new message packer
// It initially sets the message type header and system flag
func NewMsgPacker(msgType protocol.MsgType, system ...bool) *MsgPacker {
	mp := MsgPacker{}
	mp.Reset()

	systemMsg := 0
	if len(system) > 0 && system[0] {
		systemMsg = 1
	}

	mp.AddInt(int(msgType<<1) | systemMsg)

	return &mp
}
