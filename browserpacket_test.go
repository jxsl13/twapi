package main

import (
	"encoding/hex"
	"net"
	"testing"
	"time"
)

func sendBytesToMasterServerAndAwait(r []byte, t *testing.T) (responseMessage []byte) {

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

	written, err := conn.Write(r)
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("written=%d", written)

	deadline := time.Now().Add(5 * time.Second) // 90 secs timeout
	err = conn.SetReadDeadline(deadline)
	if err != nil {
		t.Error(err)
		return
	}

	buffer := make([]byte, 1500)
	readBytes, addr, err := conn.ReadFrom(buffer)
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("read=%d from %s", readBytes, addr.String())

	responseMessage = buffer[:readBytes]
	return
}

func TestTokenPacket(t *testing.T) {
	reqToken := NewTokenRequestPacket()

	response := sendBytesToMasterServerAndAwait(reqToken, t)

	token, err := NewToken(response)

	if err != nil {
		t.Error(err)
		return
	}

	srvListReq, err := NewServerListRequestPacket(token)
	if err != nil {
		t.Error(err)
		return
	}

	response = sendBytesToMasterServerAndAwait(srvListReq, t)
	t.Logf("Response: \n%s\n", hex.Dump(response[:100]))

	serverList, _, err := NewServerList(response)
	if err != nil {
		t.Error(err)
		return
	}

	t.Log(len(serverList))

	for _, srv := range serverList {
		t.Error(srv.String())
	}
	t.Error("Intended")
}

func TestServerTokenMasterServer(t *testing.T) {

}

func TestServerListToken(t *testing.T) {

}
