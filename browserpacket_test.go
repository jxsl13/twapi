package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

func TestTokenPacket(t *testing.T) {
	const expectedTokenLength = 519
	buffer := bufio.NewWriter(&bytes.Buffer{})

	browserPacket := &BrowserPacket{}
	browserPacket.token.Generate()

	written, err := io.Copy(buffer, browserPacket)

	if err != nil {
		t.Error(err)
	}

	if expectedTokenLength != written {
		t.Errorf("expected: %d written: %d bytes", expectedTokenLength, written)
	}
}

func TestServerTokenMasterServer(t *testing.T) {
	address := "master1.teeworlds.com:8283"

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
	defer conn.Close()

	_, browserPacket := NewBrowserPacket()
	io.Copy(conn, browserPacket)

	deadline := time.Now().Add(5 * time.Second) // 90 secs timeout
	err = conn.SetReadDeadline(deadline)
	if err != nil {
		return
	}

	buffer := make([]byte, 1500)
	readBytes, addr, err := conn.ReadFrom(buffer)
	if err != nil {
		return
	}
	buffer = buffer[:readBytes]

	t.Logf("Received from connection: %s", addr)
	t.Logf("Response: %s", hex.Dump(buffer))

	err = browserPacket.ParseAndSetServerToken(buffer)
	t.Logf("ClientToken: %d ServerToken: %d", browserPacket.ClientToken(), browserPacket.ServerToken())
	if err != nil {
		t.Error(err)
	}
}

func sendToMasterServerAndAwait(r io.Reader, t *testing.T) (responseMessage []byte) {

	address := "master1.teeworlds.com:8283"

	raddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		t.Errorf("NewMasterServerFromAddress: %s : %s", address, err)
		return
	}

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		t.Errorf("NewMasterServerFromAddress: %s : %s", address, err)
		return
	}
	defer conn.Close()

	io.Copy(conn, r)

	deadline := time.Now().Add(5 * time.Second) // 90 secs timeout
	err = conn.SetReadDeadline(deadline)
	if err != nil {
		t.Error(err)
		return
	}

	buffer := make([]byte, 1500)
	readBytes, _, err := conn.ReadFrom(buffer)
	if err != nil {
		t.Error(err)
		return
	}
	responseMessage = buffer[:readBytes]
	return
}

func TestServerListToken(t *testing.T) {

	// create token for handshake and corresponding server list request payload
	// tokenpacket is a member of the serverlist packet, thus they are bound together
	// as captain Picard would say: they come in pairs
	tokenPacket, serverListPacket := NewServerListPacket()

	// example of an expiration retry
	if tokenPacket.Expired() {
		tokenPacket.Generate()
		resp := sendToMasterServerAndAwait(tokenPacket, t)
		err := tokenPacket.ParseAndSetServerToken(resp)
		if err != nil {
			t.Error(err)
		}
	} else {
		t.Error("Token should be expired, as it never was renewed")
	}

	resp := sendToMasterServerAndAwait(serverListPacket, t)
	serverList, err := serverListPacket.ParseServerListResponse(resp)
	if err != nil {
		t.Error(err)
	}

	if len(serverList) == 0 {
		t.Error("Received server list is empty")
	}

	for _, addr := range serverList {
		t.Logf("Server: %s", addr.String())
	}

	t.Error("Intended")

}
