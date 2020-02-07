package browser

import (
	"bytes"
	"errors"
	"io"
)

var (
	// ErrInvalidWrite is returned if writing to an io.Writer failed
	ErrInvalidWrite = errors.New("invalid write")
)

// RequestToken writes the payload to w
func RequestToken(w io.Writer) (err error) {
	tokenReq := NewTokenRequestPacket()
	n, err := w.Write(tokenReq)
	if err != nil {
		return
	} else if n != len(tokenReq) {
		err = ErrInvalidWrite
	}

	return
}

// Request writes the payload into w.
// w can be a buffer or a udp connection
// packet can be one of:
//		"serverlist"
//		"servercount"
//		"serverinfo"
func Request(packet string, token Token, w io.Writer) (err error) {
	var payload []byte
	switch packet {
	case "serverlist":
		payload, err = NewServerListRequestPacket(token)
	case "servercount":
		payload, err = NewServerCountRequestPacket(token)
	case "serverinfo":
		payload, err = NewServerInfoRequestPacket(token)
	}
	if err != nil {
		return
	}

	n, err := w.Write(payload)
	if err != nil {
		return
	} else if n != len(payload) {
		err = ErrInvalidWrite
	}
	return
}

// MatchResponse matches a respnse to a specific string
// "", ErrInvalidResponseMessage -> if response message contains invalid data
// "", ErrInvalidHeaderLength -> if response message is too short
// "token" - token response
// "serverlist" - server list response
// "servercount" - server count response
// "serverinfo" - server info response
func MatchResponse(responseMessage []byte) (string, error) {
	if len(responseMessage) < minPrefixLength {
		return "", ErrInvalidHeaderLength
	}

	if len(responseMessage) == 12 {
		return "token", nil
	} else if bytes.Equal(sendServerListRaw, responseMessage[tokenPrefixSize:tokenPrefixSize+len(sendServerListRaw)]) {
		return "serverlist", nil
	} else if bytes.Equal(sendServerCountRaw, responseMessage[tokenPrefixSize:tokenPrefixSize+len(sendServerCountRaw)]) {
		return "servercount", nil
	} else if bytes.Equal(sendInfoRaw, responseMessage[tokenPrefixSize:tokenPrefixSize+len(sendInfoRaw)]) {
		return "serverinfo", nil
	}
	return "", ErrInvalidResponseMessage
}
