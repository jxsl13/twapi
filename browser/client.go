package browser

import (
	"bytes"
	"io"
	"time"
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

// ReceiveToken reads the token payload from the reader r
func ReceiveToken(r io.Reader) (response []byte, err error) {
	response = make([]byte, tokenResponseSize)
	read, err := r.Read(response)
	if err != nil {
		return
	}

	if read != tokenResponseSize {
		err = ErrInvalidResponseMessage
	}

	response = response[:read]
	return
}

// FetchToken tries to fetch a token from the server for a specific duration at most. a timeout below 35 ms will be set to 35 ms
func FetchToken(rwd ReadWriteDeadliner, timeout time.Duration) (response []byte, err error) {
	if timeout < minTimeout {
		timeout = minTimeout
	}

	begin := time.Now()
	timeLeft := timeout
	currentTimeout := minTimeout
	writeBurst := 1

	for {
		timeLeft = timeout - time.Now().Sub(begin)
		rwd.SetReadDeadline(time.Now().Add(currentTimeout))

		if timeLeft <= 0 {
			// early return, because timed out
			err = ErrTimeout
			return
		}

		// send multiple requests
		for i := 0; i < writeBurst; i++ {
			err = RequestToken(rwd)
			if err != nil {
				return
			}
		}

		// wait for response
		response, err = ReceiveToken(rwd)
		if err == nil {
			return
		}

		// increase time & request burst
		timeLeft = timeout - time.Now().Sub(begin)
		if timeLeft <= currentTimeout {
			currentTimeout = timeLeft
		} else {
			currentTimeout *= 2
		}
		writeBurst *= 2
	}
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

// Receive reads the response message and evaluates its validity.
// If the message is not valid it is still returned.
func Receive(packet string, r io.Reader) (response []byte, err error) {
	response = make([]byte, maxBufferSize)
	read, err := r.Read(response)
	if err != nil {
		return
	}

	if read == 0 {
		err = ErrInvalidResponseMessage
	}

	response = response[:read]
	match, err := MatchResponse(response)
	if err != nil {
		return
	}

	if match != packet {
		err = ErrRequestResponseMismatch
	}
	return
}

// Fetch is the same as Fetch, but it retries fetching data for a specific time.
func Fetch(packet string, token Token, rwd ReadWriteDeadliner, timeout time.Duration) (response []byte, err error) {
	if timeout < minTimeout {
		timeout = minTimeout
	}

	begin := time.Now()
	timeLeft := timeout
	currentTimeout := minTimeout
	writeBurst := 1

	for {
		timeLeft = timeout - time.Now().Sub(begin)
		rwd.SetReadDeadline(time.Now().Add(currentTimeout))

		if timeLeft <= 0 {
			// early return, because timed out
			err = ErrTimeout
			return
		}

		// send multiple requests
		for i := 0; i < writeBurst; i++ {
			err = Request(packet, token, rwd)
			if err != nil {
				return
			}
		}

		// wait for response
		response, err = Receive(packet, rwd)
		if err == nil {
			return
		}

		// increase time & request burst
		timeLeft = timeout - time.Now().Sub(begin)
		if timeLeft <= currentTimeout {
			currentTimeout = timeLeft
		} else {
			currentTimeout *= 2
		}
		writeBurst *= 2
	}
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
