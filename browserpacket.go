package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"time"
)

const (
	requestServerList       = "\xff\xff\xff\xffreq2"
	sendServerList          = "\xff\xff\xff\xfflis2"
	requestServerCount      = "\xff\xff\xff\xffcou2"
	sendServerCount         = "\xff\xff\xff\xffsiz2"
	tokenRefreshTimeSeconds = 90
)

// Packet contains a payload
type Packet struct {
	bytes.Buffer
}

// TokenPacket abstracts the token for the browser protocol
type TokenPacket struct {
	Packet
	clientToken int
	serverToken int
	expiresAt   time.Time
}

// ClientToken return sthe client token
func (t *TokenPacket) ClientToken() int {
	return t.clientToken
}

// ServerToken returns the server token that's being used in the follow up requests
func (t *TokenPacket) ServerToken() int {
	return t.serverToken
}

// Expired checks if the current token already expired.
func (t *TokenPacket) Expired() bool {
	timeLeft := t.expiresAt.Sub(time.Now())
	return timeLeft <= 0
}

// packs header
func packCtrlMsgWithToken(tokenServer, tokenClient int) []byte {
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

// Generate a new client token if the server token already expired.
// when a new client token is generated, the old server token expires.
func (t *TokenPacket) Generate() {

	if t.expiresAt.Sub(time.Now()) > 0 {
		return
	}

	seedSource := rand.NewSource(time.Now().UnixNano())
	randomNumberGenerator := rand.New(seedSource)

	t.clientToken = int(randomNumberGenerator.Int31())
	t.serverToken = -1

	header := packCtrlMsgWithToken(t.serverToken, t.clientToken)

	t.Reset()
	t.Write(header)
}

// RefillBuffer refills the token buffer for continuous reading
func (t *TokenPacket) RefillBuffer() {
	t.Packet.Reset()
	t.Packet.Write(packCtrlMsgWithToken(t.ServerToken(), t.ClientToken()))
}

// Len returns length of the remaining buffer size.
func (t *TokenPacket) Len() int {
	return t.Packet.Len()
}

// unpacks header
func unpackCtrlMsgWithToken(message []byte) (tokenServer, tokenClient int, err error) {
	if len(message) < 12 {
		err = fmt.Errorf("control message is too small, %d byte, required 12 byte", len(message))
		return
	}
	tokenClient = (int(message[3]) << 24) + (int(message[4]) << 16) + (int(message[5]) << 8) + int(message[6])
	tokenServer = (int(message[8]) << 24) + (int(message[9]) << 16) + (int(message[10]) << 8) + int(message[11])
	return
}

// ParseAndSetServerToken updates the server's token from a received
func (t *TokenPacket) parseAndSetServerToken(responseMessage []byte) (rest []byte, err error) {
	serverToken, clientToken, err := unpackCtrlMsgWithToken(responseMessage)
	if err != nil {
		return
	}

	// update server token in any case
	t.serverToken = serverToken

	if clientToken != t.clientToken {
		err = fmt.Errorf("client token mismatch: expected %d received: %d", t.clientToken, clientToken)
	}

	t.expiresAt = time.Now().Add(tokenRefreshTimeSeconds * time.Second)

	// return the rest after parsing
	rest = responseMessage[12:]
	return
}

// ParseAndSetServerToken updates the server's token from a received
func (t *TokenPacket) ParseAndSetServerToken(responseMessage []byte) (err error) {
	serverToken, clientToken, err := unpackCtrlMsgWithToken(responseMessage)
	if err != nil {
		return
	}

	// update server token in any case
	t.serverToken = serverToken

	if clientToken != t.clientToken {
		err = fmt.Errorf("client token mismatch: expected %d received: %d", t.clientToken, clientToken)
	}

	t.expiresAt = time.Now().Add(tokenRefreshTimeSeconds * time.Second)
	return
}

// BrowserPacket encapsulates a Teeworlds packat that is sent by the server browser.
type BrowserPacket struct {
	token          TokenPacket
	payload        Packet
	responseHeader Packet
	reads          int
}

// NewBrowserPacket creates a new browserpacket
func NewBrowserPacket() (*TokenPacket, *BrowserPacket) {
	bp := BrowserPacket{}
	bp.token.Generate()

	return &bp.token, &bp
}

// Token returns a pointer to the underlying token
// this is used in order to further check if the token already expired.
func (bp *BrowserPacket) Token() *TokenPacket {
	return &bp.token
}

// Reset resets the state to allow for another read
func (bp *BrowserPacket) Reset() {
	bp.reads = 0
	bp.token.Reset()
	bp.payload.Reset()
}

// Read data from token header and payload.
func (bp *BrowserPacket) Read(p []byte) (n int, err error) {
	if bp.token.Expired() {
		return 0, errors.New("token expired")
	}
	if bp.reads == 0 && bp.token.Len() == 0 {

	}

	tokenSize, err1 := bp.token.Read(p)
	payloadSize, err2 := bp.payload.Read(p)
	bp.reads++

	if err1 == io.EOF && err2 == io.EOF {
		return 0, io.EOF
	}

	n = tokenSize + payloadSize
	return
}

// Write data into payload.
func (bp *BrowserPacket) Write(p []byte) (n int, err error) {
	n, err = bp.payload.Write(p)
	return
}

// ParseAndSetServerToken updates the internal token timeout as well as the internal server token that is used to
// secure the connection. The token allows for further requests that are not solely the connection.
func (bp *BrowserPacket) ParseAndSetServerToken(responseMessage []byte) (err error) {
	err = bp.token.ParseAndSetServerToken(responseMessage)
	return
}

// ClientToken returns the client token
func (bp *BrowserPacket) ClientToken() int {
	return bp.token.ClientToken()
}

// ServerToken returns the server token
func (bp *BrowserPacket) ServerToken() int {
	return bp.token.ServerToken()
}

func packTokenHeader(tokenClient, tokenServer int, packetConstant string) (header []byte) {
	const netPacketFlagConnless = 8
	const netPacketVersion = 1

	binaryPacketConstant := []byte(packetConstant)
	header = make([]byte, 9, 9+len(binaryPacketConstant))

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

// AddPayloadConstant adds a request header to the Packet
func (bp *BrowserPacket) addPayloadConstant(packetConstant string) {
	bp.payload.Write(packTokenHeader(bp.ClientToken(), bp.ServerToken(), packetConstant))
}

// AddResponseConstant initializes the expected response header
func (bp *BrowserPacket) addResponseConstant(packetConstant string) {
	bp.responseHeader.Write(packTokenHeader(bp.ServerToken(), bp.ClientToken(), packetConstant))
}

// AddResponseConstant initializes the expected response header
func (bp *BrowserPacket) responseHeaderOk(responseMessage []byte) (data []byte, ok bool) {
	expectedLen := bp.responseHeader.Len()

	if len(responseMessage) < expectedLen {
		data = responseMessage
		ok = false
		return
	}
	receivedHeader := responseMessage[:expectedLen]

	if bytes.Compare(receivedHeader, bp.responseHeader.Bytes()) == 0 {
		ok = true
	}

	data = responseMessage[expectedLen:]
	return

}

// ServerListPacket is used to request the server list from the master server
type ServerListPacket struct {
	*BrowserPacket
}

// NewServerListPacket returns a new server list request packet
func NewServerListPacket() (*TokenPacket, *ServerListPacket) {

	tp, bp := NewBrowserPacket()
	slp := &ServerListPacket{bp}

	slp.addPayloadConstant(requestServerList)
	slp.addResponseConstant(sendServerList)
	return tp, slp
}

// ParseServerListResponse returns a server list from
func (slp *ServerListPacket) ParseServerListResponse(responseMessage []byte) (serverList []net.UDPAddr, err error) {
	// create buffer for server addresses
	serverList = make([]net.UDPAddr, 0, 1)

	// parse received handshake token
	rest, err := slp.BrowserPacket.token.parseAndSetServerToken(responseMessage)

	// parse header and compare with expected header
	data, ok := slp.BrowserPacket.responseHeaderOk(rest)
	if !ok {
		err = errors.New("response header mismatch")
		return
	}

	// parse server list
	numServers := len(data) / 18 // 18 byte, 16 for IPv4/IPv6 and 2 bytes for the port
	serverList = make([]net.UDPAddr, 0, numServers)

	// fixed size array
	ipv4Prefix := [12]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF}

	for idx := 0; idx < numServers; idx++ {
		ip := []byte{}

		// compare byte slices
		if bytes.Equal(ipv4Prefix[:], data[idx*18:idx*18+12]) {
			// IPv4 has a prefix
			ip = data[idx*18+12 : idx*18+16]
		} else {
			// full IP is the IPv6 otherwise
			ip = data[idx*18 : idx*18+16]
		}

		serverList = append(serverList, net.UDPAddr{
			IP:   ip,
			Port: (int(data[idx*18+16]) << 8) + int(data[idx*18+17]),
		})
	}
	return
}
