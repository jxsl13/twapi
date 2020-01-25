package main

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"time"
)

const (
	getList                   = "\xff\xff\xff\xffreq2"
	receiveList               = "\xff\xff\xff\xfflis2"
	getCount                  = "\xff\xff\xff\xffcou2"
	receiveCount              = "\xff\xff\xff\xffsiz2"
	tokenRefreshTimeInSeconds = 90
)

// MasterServer represents a masterserver
// Info: Do not forget to call defer Close() after creating the MasterServer
type MasterServer struct {
	*BrowserConnection
}

// Implements the stringer interface
func (ms *MasterServer) String() string {
	ms.RLock()
	defer ms.RUnlock()

	return fmt.Sprintf("Masterserver: %s", ms.RemoteAddr())
}

// NewMasterServerFromAddress creates a new MasterServer struct
// that that creates an internal udp connection.
// Info: Do not forget to call defer Close() after creating the MasterServer
func NewMasterServerFromAddress(address string) (ms MasterServer, err error) {

	raddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		err = fmt.Errorf("NewMasterServerFromAddress: %s : %s", address, err)
		return
	}

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		err = fmt.Errorf("NewMasterServerFromAddress: %s : %s", address, err)
		return
	}

	ms, err = NewMasterServerFromUDPConn(conn)
	return
}

// NewMasterServerFromUDPConn creates a new MasterServer that
// uses an existing udp connection in order to share it multithreaded
func NewMasterServerFromUDPConn(conn *net.UDPConn) (ms MasterServer, err error) {
	if conn == nil {
		err = errors.New("NewMasterServerFromUDPConn: nil UDPConn passed")
		return
	}

	bc, err := NewBrowserConnetion(conn, tokenRefreshTimeInSeconds*time.Second)
	if err != nil {
		err = fmt.Errorf("NewMasterServerFromUDPConn: %s : %s", conn.RemoteAddr(), err.Error())
		return
	}

	// embed the newly created connection
	ms.BrowserConnection = &bc
	return
}

// GetServerList retrieves the server list from the masterserver
func (ms *MasterServer) GetServerList() (serverList []net.UDPAddr, err error) {

	data, err := ms.Request(getList, receiveList, 75*16*18)
	if err != nil {
		return
	}
	/*
		each server information contains of 18 bytes
		first 16 bytes define the IP
		the last 2 bytes define the port

		if the first 12 bytes match the defined pefix, the IP is parsed as IPv4
		and if it does not match, the IP is parsed as IPv6
	*/
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

// GetServerCount Requests the number of currently registered servers at the master server.
func (ms *MasterServer) GetServerCount() (count int, err error) {

	data, err := ms.Request(getCount, receiveCount, 64)
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
