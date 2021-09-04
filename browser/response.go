package browser

import (
	"bytes"
	"net"
)

func NewResponsePacket(data []byte) (*ResponsePacket, error) {
	res := &ResponsePacket{}
	err := res.UnmarshalBinary(data)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// ResponsePacket is a response that contains a token, a matched response header in
// order to identify that type and the ramaining payload that contains the data
type ResponsePacket struct {
	Addr           *net.UDPAddr
	Token          Token
	ResponseHeader string
	Payload        []byte
}

func (rp *ResponsePacket) UnmarshalBinary(data []byte) error {
	if len(data) == tokenResponseSize {
		rp.Token.UnmarshalBinary(data)
		rp.ResponseHeader = "token"
		rp.Payload = make([]byte, 0)
	}

	// parse token prefix
	err := rp.Token.UnmarshalBinary(data)
	if err != nil {
		return err
	}
	// skip token part
	data = data[tokenPrefixSize:]

	// try matching all known prefix values
	// in order to set the header value
	// TODO: externalize this and allow to register custom prefixes
	offset := 0
	for _, prefix := range ResponsePacketList {
		if isBytePrefix(prefix, data) {
			offset = len(prefix)
			rp.ResponseHeader = string(prefix)
			break
		}
	}
	rp.Payload = data[offset:]
	return nil
}

// is prefix
func isBytePrefix(prefix, data []byte) bool {
	if len(data) < len(prefix) {
		return false
	}
	return bytes.Equal(prefix, data[:len(prefix)])
}
