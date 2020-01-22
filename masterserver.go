package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"sync/atomic"
	"time"
)

const (
	getList   = "\xff\xff\xff\xffreq2"
	sendList  = "\xff\xff\xff\xfflis2"
	getCount  = "\xff\xff\xff\xffcou2"
	sendCount = "\xff\xff\xff\xffsiz2"
	getInfo   = "\xff\xff\xff\xffgie2"
	sendInfo  = "\xff\xff\xff\xffinf3"
)

// MasterServer represents a masterserver
// Info: Do not forget to call defer Close() after creating the MasterServer
type MasterServer struct {
	*net.UDPConn               // embed type in order to work on the MasterServer like on a socket.
	tokenClient  uint32        // token that I as the client created
	tokenServer  uint32        // token that the master server created
	timeout      time.Duration // timeout to before timing out on a receiving socket
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
	ms.timeout = 5 * time.Second
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
	ms.timeout = 5 * time.Second

	return
}

// SetTimeout sets the duration before a receiving socket times out
func (ms *MasterServer) SetTimeout(timeout time.Duration) {
	ms.timeout = timeout
}

func (ms *MasterServer) request(req []byte, responseSize int) (resp []byte, err error) {
	data := bytes.NewBuffer(req)
	conn := ms.UDPConn // alias

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

	// do not pass empty bytes
	resp = buffer[:readBytes]
	return
}

// get the request header
func (ms *MasterServer) getHeaderToSend(packetConstant string) (header []byte) {
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

// get the request header
func (ms *MasterServer) getHeaderToReceive(packetConstant string) (header []byte) {
	const netPacketFlagConnless = 8
	const netPacketVersion = 1

	binaryPacketConstant := []byte(packetConstant)
	header = make([]byte, 9, 9+len(binaryPacketConstant))

	// access members threadsafe
	tokenClient := atomic.LoadUint32(&ms.tokenClient)
	tokenServer := atomic.LoadUint32(&ms.tokenServer)

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

// GetServerList retrieves the server list from the masterserver
func (ms *MasterServer) GetServerList() (serverList []net.UDPAddr, err error) {

	err = ms.refreshToken()
	if err != nil {
		err = fmt.Errorf("%s : %s", "failed to refresh token handshake", err)
		return
	}

	headerToSend := ms.getHeaderToSend(getList)

	responseData, err := ms.request(headerToSend, 4096)
	if err != nil {
		return
	}

	// client and server token are inverted in this case
	headerToReceive := ms.getHeaderToReceive(sendList)

	responseHeader := responseData[:len(headerToReceive)]

	if !bytes.Equal(headerToReceive, responseHeader) {
		err = fmt.Errorf("Expexted header: \n%s\nReceived header:\n%s", hex.Dump(headerToReceive), hex.Dump(responseHeader))
		return
	}

	data := responseData[len(headerToReceive):]
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

// RefreshToken updates the needed token for a safe communication
// todo: check export/visibility
func (ms *MasterServer) refreshToken() (err error) {

	packControlMessageWithToken := func(tokenServer, tokenClient int32) []byte {
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

	unpackControlMessageWithToken := func(message []byte) (tokenServer, tokenClient uint32, err error) {
		if len(message) < 12 {
			err = fmt.Errorf("control message is too small, %d byte, required 12 byte", len(message))
			return
		}
		tokenClient = (uint32(message[3]) << 24) + (uint32(message[4]) << 16) + (uint32(message[5]) << 8) + uint32(message[6])
		tokenServer = (uint32(message[8]) << 24) + (uint32(message[9]) << 16) + (uint32(message[10]) << 8) + uint32(message[11])
		return
	}

	seedSource := rand.NewSource(time.Now().UnixNano())
	randomNumberGenerator := rand.New(seedSource)

	tokenClientGenerated := randomNumberGenerator.Int31()
	atomic.StoreUint32(&ms.tokenClient, uint32(tokenClientGenerated))

	toSend := packControlMessageWithToken(-1, tokenClientGenerated)
	received, err := ms.request(toSend, 16) // expecting 12 bytes at max
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

	atomic.StoreUint32(&ms.tokenServer, tokenServerReceived)
	fmt.Printf("token-refresh: client=%d server=%d\n", tokenClientReceived, tokenServerReceived)

	return
}
