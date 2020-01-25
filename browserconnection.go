package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	loggingEnabled = false
)

// BrowserConnection is an abstraction of what the teeworlds ingame browser can do
// request data either from the master servers of from a Teeworlds server
type BrowserConnection struct {
	*net.UDPConn                  // embed type in order to work on the MasterServer like on a socket.
	sync.RWMutex                  // use this mutex when modifying data
	tokenClient     uint32        // token that I as the client created
	tokenServer     uint32        // token that the master server created
	timeout         time.Duration // timeout to before timing out on a receiving socket
	tokenExpiration time.Time     // time after which a new token needs to be requested.
}

// NewBrowserConnetion creates a new browser connection from an existing udp connection
func NewBrowserConnetion(conn *net.UDPConn, timeout time.Duration) (bc BrowserConnection, err error) {
	if conn == nil {
		err = errors.New("BrowserConnection: passed UDPConn is nil")
		return
	}

	bc.timeout = timeout
	bc.tokenExpiration = time.Now().Add(-1 * time.Second) // already expired
	bc.UDPConn = conn
	return
}

// Timeout Set the timeout to a new value
func (bc *BrowserConnection) Timeout(timeout time.Duration) {
	bc.Lock()
	defer bc.Unlock()

	bc.timeout = timeout
}

// Request sends a specific request header and expects a specific response header with a specific response size.
// if the request succeeds, resp contains the response data without the expected response header.
func (bc *BrowserConnection) Request(sentHeader, expectedResponseHeader string, responseSize int) (resp []byte, err error) {

	// TODO: set timeout to 10ms and increase is by (10 ms * 2^n) also increase the burst size of packets
	err = bc.refreshToken()
	if err != nil {
		err = fmt.Errorf("%s : %s", "failed to refresh token handshake", err)
		return
	}

	headerToSend := bc.packHeadertoSend(sentHeader)

	data := bytes.NewBuffer(headerToSend)
	conn := bc.UDPConn // alias, is threadsafe on its own

	writtenBytes, err := io.Copy(conn, data)
	if err != nil {
		return
	}

	sentTo := conn.RemoteAddr().String()
	if loggingEnabled {
		fmt.Printf("request-sent: bytes=%d to=%s\n", writtenBytes, sentTo)
	}

	// will contain response data
	buffer := make([]byte, responseSize)

	deadline := time.Now().Add(bc.timeout) // 90 secs timeout
	err = conn.SetReadDeadline(deadline)
	if err != nil {
		return
	}

	readBytes, addr, err := conn.ReadFrom(buffer)
	if err != nil {
		return
	}

	receivedFrom := addr.String()
	if receivedFrom != sentTo {
		err = fmt.Errorf("received data from wrong entity: expected=%s gotten=%s", sentTo, receivedFrom)
		return
	}
	if loggingEnabled {
		fmt.Printf("response-received: bytes=%d from=%s\n", readBytes, receivedFrom)
	}

	// complete response data without pending zeroes
	responseData := buffer[:readBytes]

	// client and server token are inverted in this case
	headerToReceive := bc.packHeaderToReceive(expectedResponseHeader)
	responseHeader := responseData[:len(headerToReceive)]

	if !bytes.Equal(headerToReceive, responseHeader) {
		err = fmt.Errorf("Expected header: \n%s\nReceived header:\n%s", hex.Dump(headerToReceive), hex.Dump(responseHeader))
		return
	}

	// return data without the header or pending bytes
	resp = responseData[len(headerToReceive):]
	return
}

// get the request header
func (bc *BrowserConnection) packHeadertoSend(packetConstant string) (header []byte) {
	const netPacketFlagConnless = 8
	const netPacketVersion = 1

	binaryPacketConstant := []byte(packetConstant)
	header = make([]byte, 9, 9+len(binaryPacketConstant))

	// access members threadsafe
	tokenClient := atomic.LoadUint32(&bc.tokenClient)
	tokenServer := atomic.LoadUint32(&bc.tokenServer)

	// Header
	header[0] = ((netPacketFlagConnless << 2) & 0b11111100) | (netPacketVersion & 0b00000011)
	header[1] = byte(tokenServer >> 24)
	header[2] = byte(tokenServer >> 16)
	header[3] = byte(tokenServer >> 8)
	header[4] = byte(tokenServer)
	// ResponseToken
	header[5] = byte(tokenClient >> 24)
	header[6] = byte(tokenClient >> 16)
	header[7] = byte(tokenClient >> 8)
	header[8] = byte(tokenClient)

	header = append(header, binaryPacketConstant...)
	return
}

// get the expected response header
func (bc *BrowserConnection) packHeaderToReceive(packetConstant string) (header []byte) {
	const netPacketFlagConnless = 8
	const netPacketVersion = 1

	binaryPacketConstant := []byte(packetConstant)
	header = make([]byte, 9, 9+len(binaryPacketConstant))

	// access members threadsafe
	bc.RLock()
	tokenClient := bc.tokenClient
	tokenServer := bc.tokenServer
	bc.RUnlock()

	// Header
	header[0] = ((netPacketFlagConnless << 2) & 0b11111100) | (netPacketVersion & 0b00000011)
	header[1] = byte(tokenClient >> 24)
	header[2] = byte(tokenClient >> 16)
	header[3] = byte(tokenClient >> 8)
	header[4] = byte(tokenClient)
	// ResponseToken
	header[5] = byte(tokenServer >> 24)
	header[6] = byte(tokenServer >> 16)
	header[7] = byte(tokenServer >> 8)
	header[8] = byte(tokenServer)

	header = append(header, binaryPacketConstant...)
	return
}

// RefreshToken updates the handshake token if it has not yet expired
func (bc *BrowserConnection) refreshToken() (err error) {

	// check if a refresh is needed
	now := time.Now()

	bc.RLock()
	expirationTime := bc.tokenExpiration
	bc.RUnlock()

	if now.Sub(expirationTime) <= 0 {
		// no refresh needed
		return
	}

	// needs a refresh
	bc.Lock()
	bc.tokenExpiration = now.Add(tokenRefreshTimeInSeconds * time.Second)
	bc.Unlock()

	seedSource := rand.NewSource(time.Now().UnixNano())
	randomNumberGenerator := rand.New(seedSource)

	tokenClientGenerated := randomNumberGenerator.Int31()
	bc.Lock()
	bc.tokenClient = uint32(tokenClientGenerated)
	bc.Unlock()

	toSend := packControlMessageWithToken(-1, tokenClientGenerated)
	received, err := bc.sendRequest(toSend, 16) // expecting 12 bytes at max
	if err != nil {
		return
	}

	tokenServerReceived, tokenClientReceived, err := unpackControlMessageWithToken(received)
	if err != nil {
		return
	}

	tokenClientGeneratedUint32 := uint32(tokenClientGenerated)
	if tokenClientReceived != tokenClientGeneratedUint32 {
		err = fmt.Errorf("client token mismatch: generated_token=%d received_token=%d", tokenClientGeneratedUint32, tokenClientReceived)
		return
	}

	bc.Lock()
	bc.tokenServer = tokenServerReceived
	bc.Unlock()

	if loggingEnabled {
		fmt.Printf("token-refresh: client=%d server=%d\n", tokenClientReceived, tokenServerReceived)
	}
	return
}

// packs header
func packControlMessageWithToken(tokenServer, tokenClient int32) []byte {
	const netPacketFlagControl = 1
	const netControlMessageToken = 5
	const netTokenRequestDataSize = 512

	const size = 4 + 3 + netTokenRequestDataSize
	b := make([]byte, size)

	// Header
	b[0] = (netPacketFlagControl << 2) & 0b11111100
	b[3] = byte(tokenServer >> 24)
	b[4] = byte(tokenServer >> 16)
	b[5] = byte(tokenServer >> 8)
	b[6] = byte(tokenServer)
	// Data
	b[7] = netControlMessageToken
	b[8] = byte(tokenClient >> 24)
	b[9] = byte(tokenClient >> 16)
	b[10] = byte(tokenClient >> 8)
	b[11] = byte(tokenClient)
	return b
}

// unpacks header
func unpackControlMessageWithToken(message []byte) (tokenServer, tokenClient uint32, err error) {
	if len(message) < 12 {
		err = fmt.Errorf("control message is too small, %d byte, required 12 byte", len(message))
		return
	}
	tokenClient = (uint32(message[3]) << 24) + (uint32(message[4]) << 16) + (uint32(message[5]) << 8) + uint32(message[6])
	tokenServer = (uint32(message[8]) << 24) + (uint32(message[9]) << 16) + (uint32(message[10]) << 8) + uint32(message[11])
	return
}

// specific function for requesting th handshake token
func (bc *BrowserConnection) sendRequest(req []byte, responseSize int) (resp []byte, err error) {
	data := bytes.NewBuffer(req)
	conn := bc.UDPConn // alias, threadsafe

	writtenBytes, err := io.Copy(conn, data)
	if err != nil {
		return
	}
	sentTo := conn.RemoteAddr().String()
	if loggingEnabled {
		fmt.Printf("request-token: bytes=%d to=%s\n", writtenBytes, sentTo)
	}

	// will contain response data
	buffer := make([]byte, responseSize)

	deadline := time.Now().Add(bc.timeout) // 5 secs timeout
	err = conn.SetReadDeadline(deadline)
	if err != nil {
		return
	}

	readBytes, addr, err := conn.ReadFrom(buffer)
	if err != nil {
		return
	}

	receivedFrom := addr.String()
	if receivedFrom != sentTo {
		err = fmt.Errorf("received data from wrong entity: expected=%s gotten=%s", sentTo, receivedFrom)
		return
	}
	if loggingEnabled {
		fmt.Printf("response-token: bytes=%d from=%s\n", readBytes, receivedFrom)
	}

	// do not pass empty bytes
	resp = buffer[:readBytes]
	return
}
