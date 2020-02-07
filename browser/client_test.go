package browser

import (
	"net"
	"testing"
	"time"
)

const (
	doTimedTests = false // tests with timeouts
)

func sendBytesToServer(r []byte, raddr net.UDPAddr, t *testing.T) (addr net.Addr, responseMessage []byte) {
	conn, err := net.DialUDP("udp", nil, &raddr)
	if err != nil {
		t.Errorf("NewMasterServerFromAddress: %s : %s", raddr.String(), err)
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

	buffer := make([]byte, 1500*16) // max payload of a tw server is 1400
	readBytes, addr, err := conn.ReadFrom(buffer)
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("read=%d from %s", readBytes, addr.String())

	responseMessage = buffer[:readBytes]
	return
}

func sendBytesToMasterServerAndAwait(r []byte, t *testing.T) (addr net.Addr, responseMessage []byte) {

	address := "master1.teeworlds.com:8283"

	raddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		t.Errorf("NewMasterServerFromAddress: %s : %s", address, err)
		return
	}

	addr, responseMessage = sendBytesToServer(r, *raddr, t)
	return
}

func TestTokenPacketAndServerList(t *testing.T) {
	reqToken := NewTokenRequestPacket()

	_, response := sendBytesToMasterServerAndAwait(reqToken, t)

	match, err := MatchResponse(response)
	if err != nil {
		t.Error(err)
		return
	}

	if match != "token" {
		t.Errorf("invalid match: %s", match)
		return
	}

	token, err := ParseToken(response)

	t.Log(token.String())

	if err != nil {
		t.Error(err)
		return
	}

	srvListReq, err := NewServerListRequestPacket(token)
	if err != nil {
		t.Error(err)
		return
	}

	if doTimedTests {
		// retrieved token, token expires in 16 seconds
		time.Sleep(16 * time.Second)
	}

	_, response = sendBytesToMasterServerAndAwait(srvListReq, t)

	if (len(response)-9-8)%18 != 0 {
		t.Errorf("invalid response length for server list: %d", len(response))
	}

	match, err = MatchResponse(response)
	if err != nil {
		t.Error(err)
		return
	}

	if match != "serverlist" {
		t.Errorf("invalid match: %s", match)
		return
	}

	serverList, err := ParseServerList(response)
	if err != nil {
		t.Error(err)
		return
	}

	t.Logf("Servers retrieved: %d\n", len(serverList))
}

func TestServerCount(t *testing.T) {
	reqToken := NewTokenRequestPacket()

	_, response := sendBytesToMasterServerAndAwait(reqToken, t)

	token, err := ParseToken(response)

	if err != nil {
		t.Error(err)
		return
	}

	srvCountReq, err := NewServerCountRequestPacket(token)
	if err != nil {
		t.Error(err)
		return
	}

	_, response = sendBytesToMasterServerAndAwait(srvCountReq, t)
	count, err := ParseServerCount(response)
	if err != nil {
		t.Error(err)
		return
	}

	match, err := MatchResponse(response)
	if err != nil {
		t.Error(err)
		return
	}

	if match != "servercount" {
		t.Errorf("invalid match: %s", match)
		return
	}

	t.Logf("Server count: %d", count)
}

func TestServerInfo(t *testing.T) {
	reqToken := NewTokenRequestPacket()

	_, response := sendBytesToMasterServerAndAwait(reqToken, t)

	token, err := ParseToken(response)

	if err != nil {
		t.Error(err)
		return
	}

	srvListReq, err := NewServerListRequestPacket(token)
	if err != nil {
		t.Error(err)
		return
	}

	_, response = sendBytesToMasterServerAndAwait(srvListReq, t)

	serverList, err := ParseServerList(response)
	if err != nil {
		t.Error(err)
		return
	}

	reqToken = NewTokenRequestPacket()

	_, response = sendBytesToServer(reqToken, serverList[0], t)
	token, err = ParseToken(response)
	if err != nil {
		t.Error(err)
		return
	}

	srvInfoReq, err := NewServerInfoRequestPacket(token)
	if err != nil {
		t.Error(err)
		return
	}

	addr, response := sendBytesToServer(srvInfoReq, serverList[0], t)
	serverInfo, err := ParseServerInfo(response, addr)
	if err != nil {
		t.Error(err)
		return
	}

	match, err := MatchResponse(response)
	if err != nil {
		t.Error(err)
		return
	}

	if match != "serverinfo" {
		t.Errorf("invalid match: %s", match)
		return
	}

	t.Logf("Server count: %s", serverInfo.String())
}
