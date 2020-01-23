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
	getList                   = "\xff\xff\xff\xffreq2"
	sendList                  = "\xff\xff\xff\xfflis2"
	getCount                  = "\xff\xff\xff\xffcou2"
	sendCount                 = "\xff\xff\xff\xffsiz2"
	getInfo                   = "\xff\xff\xff\xffgie3"
	sendInfo                  = "\xff\xff\xff\xffinf3"
	tokenRefreshTimeInSeconds = 90
)

// MasterServer represents a masterserver
// Info: Do not forget to call defer Close() after creating the MasterServer
type MasterServer struct {
	*net.UDPConn                  // embed type in order to work on the MasterServer like on a socket.
	sync.RWMutex                  // use this mutex when modifying data
	tokenClient     uint32        // token that I as the client created
	tokenServer     uint32        // token that the master server created
	timeout         time.Duration // timeout to before timing out on a receiving socket
	tokenExpiration time.Time     // time after which a new token needs to be requested.
}

// Implements the stringer interface
func (ms *MasterServer) String() string {
	ms.RLock()
	defer ms.RUnlock()

	return fmt.Sprintf("Masterserver: %s", ms.RemoteAddr().String())
}

// NewMasterServerFromAddress creates a new MasterServer struct
// that that creates an internal udp connection.
// Info: Do not forget to call defer Close() after creating the MasterServer
func NewMasterServerFromAddress(address string) (ms MasterServer, err error) {

	raddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return
	}

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return
	}

	// embed the newly created connection
	ms.UDPConn = conn
	ms.timeout = tokenRefreshTimeInSeconds * time.Second
	ms.tokenExpiration = time.Now().Add(-1 * time.Second) // already expired

	return
}

// NewMasterServerFromUDPConn creates a new MasterServer that
// uses an existing udp connection in order to share it multithreaded
func NewMasterServerFromUDPConn(conn *net.UDPConn) (ms MasterServer, err error) {
	if conn == nil {
		err = errors.New("nil UDPConn passed")
		return
	}

	// embed connection into masterserver
	ms.UDPConn = conn
	ms.timeout = tokenRefreshTimeInSeconds * time.Second
	ms.tokenExpiration = time.Now().Add(-1 * time.Second) // already expired.

	return
}

// SetTimeout sets the duration before a receiving socket times out
func (ms *MasterServer) SetTimeout(timeout time.Duration) {
	ms.Lock()
	defer ms.Unlock()

	ms.timeout = timeout
}

// GetServerList retrieves the server list from the masterserver
func (ms *MasterServer) GetServerList() (serverList []net.UDPAddr, err error) {

	data, err := ms.request(getList, sendList, 4096)
	if err != nil {
		return
	}

	numServers := len(data) / 18 // 18 byte, 16 for IPv4/IPv6 and 2 bytes for the port
	serverList = make([]net.UDPAddr, 0, numServers)

	for idx := 0; idx < numServers; idx++ {

		serverList = append(serverList, net.UDPAddr{
			IP:   data[idx*18 : idx*18+16],
			Port: (int(data[idx*18+16]) << 8) + int(data[idx*18+17]),
		})
	}
	return
}

// GetServerCount requests the number of currently registered servers at the master server.
func (ms *MasterServer) GetServerCount() (count int, err error) {

	data, err := ms.request(getCount, sendCount, 64)
	if err != nil {
		return
	}

	// should not happen
	const bytesInReturnValue = 4
	if len(data) > bytesInReturnValue {
		data = data[len(data)-bytesInReturnValue:]
	}

	for idx, b := range data {
		count |= (int(b) << (len(data) - idx))
	}

	return
}

func (ms *MasterServer) request(sentHeader, expectedResponseHeader string, responseSize int) (resp []byte, err error) {

	err = ms.refreshToken()
	if err != nil {
		err = fmt.Errorf("%s : %s", "failed to refresh token handshake", err)
		return
	}

	headerToSend := ms.packHeadertoSend(sentHeader)

	data := bytes.NewBuffer(headerToSend)
	conn := ms.UDPConn // alias, is threadsafe on its own

	writtenBytes, err := io.Copy(conn, data)
	if err != nil {
		return
	}
	sentTo := conn.RemoteAddr().String()
	fmt.Printf("request-sent: bytes=%d to=%s\n", writtenBytes, sentTo)

	// will contain response data
	buffer := make([]byte, responseSize)

	deadline := time.Now().Add(ms.timeout) // 5 secs timeout
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
	fmt.Printf("response-received: bytes=%d from=%s\n", readBytes, receivedFrom)

	// complete response data without pending zeroes
	responseData := buffer[:readBytes]

	// client and server token are inverted in this case
	headerToReceive := ms.packHeaderToReceive(expectedResponseHeader)
	responseHeader := responseData[:len(headerToReceive)]

	if !bytes.Equal(headerToReceive, responseHeader) {
		err = fmt.Errorf("Expexted header: \n%s\nReceived header:\n%s", hex.Dump(headerToReceive), hex.Dump(responseHeader))
		return
	}

	// return data without the header or pending bytes
	resp = responseData[len(headerToReceive):]
	return
}

// get the request header
func (ms *MasterServer) packHeadertoSend(packetConstant string) (header []byte) {
	const netPacketFlagConnless = 8
	const netPacketVersion = 1

	binaryPacketConstant := []byte(packetConstant)
	header = make([]byte, 9, 9+len(binaryPacketConstant))

	// access members threadsafe
	tokenClient := atomic.LoadUint32(&ms.tokenClient)
	tokenServer := atomic.LoadUint32(&ms.tokenServer)

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
func (ms *MasterServer) packHeaderToReceive(packetConstant string) (header []byte) {
	const netPacketFlagConnless = 8
	const netPacketVersion = 1

	binaryPacketConstant := []byte(packetConstant)
	header = make([]byte, 9, 9+len(binaryPacketConstant))

	// access members threadsafe
	ms.RLock()
	tokenClient := ms.tokenClient
	tokenServer := ms.tokenServer
	ms.RUnlock()

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
func (ms *MasterServer) refreshToken() (err error) {

	// check if a refresh is needed
	now := time.Now()

	ms.RLock()
	expirationTime := ms.tokenExpiration
	ms.RUnlock()

	if now.Sub(expirationTime) <= 0 {
		// no refresh needed
		return
	}

	// needs a refresh
	ms.Lock()
	ms.tokenExpiration = now.Add(tokenRefreshTimeInSeconds * time.Second)
	ms.Unlock()

	seedSource := rand.NewSource(time.Now().UnixNano())
	randomNumberGenerator := rand.New(seedSource)

	tokenClientGenerated := randomNumberGenerator.Int31()
	ms.Lock()
	ms.tokenClient = uint32(tokenClientGenerated)
	ms.Unlock()

	toSend := packControlMessageWithToken(-1, tokenClientGenerated)
	received, err := ms.sendRequest(toSend, 16) // expecting 12 bytes at max
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

	ms.Lock()
	ms.tokenServer = tokenServerReceived
	ms.Unlock()

	fmt.Printf("token-refresh: client=%d server=%d\n", tokenClientReceived, tokenServerReceived)
	return
}

// packs header
func packControlMessageWithToken(tokenServer, tokenClient int32) []byte {
	const netPacketFlagControl = 1
	const netControlMessageToken = 5
	const netTokenRequestDataSize = 512

	const size = 4 + 3 + netTokenRequestDataSize
	b := make([]byte, size, size)

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

// specific function for requesting th handshake token
func (ms *MasterServer) sendRequest(req []byte, responseSize int) (resp []byte, err error) {
	data := bytes.NewBuffer(req)
	conn := ms.UDPConn // alias, threadsafe

	writtenBytes, err := io.Copy(conn, data)
	if err != nil {
		return
	}
	sentTo := conn.RemoteAddr().String()
	fmt.Printf("request-token: bytes=%d to=%s\n", writtenBytes, sentTo)

	// will contain response data
	buffer := make([]byte, responseSize)

	deadline := time.Now().Add(ms.timeout) // 5 secs timeout
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
	fmt.Printf("response-token: bytes=%d from=%s\n", readBytes, receivedFrom)

	// do not pass empty bytes
	resp = buffer[:readBytes]
	return
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
