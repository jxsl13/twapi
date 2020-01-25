package main

import (
	"net"
	"testing"
	"time"
)

func TestGetServerList(t *testing.T) {
	ms, err := NewMasterServerFromAddress("master1.teeworlds.com:8283")
	if err != nil {
		t.Errorf("Failed to create a masterserver: %s", err)
		return
	}
	// disconnect from masterserver and close socket when leaving the test
	defer ms.Close()

	servers, err := ms.GetServerList()
	if err != nil {
		t.Error(err)
	}

	empty := net.UDPAddr{}

	for _, server := range servers {
		if server.Port == empty.Port || server.IP.Equal(empty.IP) {
			t.Errorf("Found empty server address: %s:%d", server.IP, server.Port) // re don't want empty data, that is part of the buffer
		}
	}
}

func TestAllMasterServers(t *testing.T) {
	masterServers := getMasterServerList()
	servers := make([]net.UDPAddr, 0, 1024)

	for _, ms := range masterServers {
		defer ms.Close()
	}

	for _, ms := range masterServers {
		list, err := ms.GetServerList()
		if err != nil {
			t.Errorf("error retrieving server list from masterserver %s: %s", ms.UDPConn.RemoteAddr(), err)
		} else {
			servers = append(servers, list...)
		}
	}
}

func TestTokenRefresh(t *testing.T) {

	if !doTimedTests {
		return
	}

	ms, err := NewMasterServerFromAddress("master1.teeworlds.com:8283")
	if err != nil {
		t.Errorf("Failed to create a masterserver: %s", err)
		return
	}
	// disconnect from masterserver and close socket when leaving the test
	defer ms.Close()

	err = ms.refreshToken()
	if err != nil {
		t.Error(err)
		return
	}

	ms.Timeout(100 * time.Second)
	time.Sleep(95 * time.Second)

	_, err = ms.sendRequest(ms.packHeadertoSend(getList), 4096)

	if err == nil {
		t.Error("Expecting i/o timeout because no refresh token was sent")
		return
	}
}

func TestGetServerCount(t *testing.T) {
	masterservers := getMasterServerList()

	if len(masterservers) == 0 {
		t.Errorf("Could not create enough master servers for testing: %d", len(masterservers))
		return
	}
	for _, ms := range masterservers {
		serverCount, err := ms.GetServerCount()
		if err != nil {
			t.Error(err)
			return
		}

		if serverCount == 0 || serverCount > 75*16 {
			t.Errorf("Server count: %d", serverCount)
		}
	}
}
