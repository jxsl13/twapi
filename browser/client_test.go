package browser

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type asyncCounter int64

func (ac *asyncCounter) Inc() {
	atomic.AddInt64((*int64)(ac), 1)
}

func (ac *asyncCounter) String() string {
	value := atomic.LoadInt64((*int64)(ac))
	return fmt.Sprintf("%d", value)
}

func getServerInfo(addr *net.UDPAddr, t *testing.T, wg *sync.WaitGroup, cnt *asyncCounter) {
	defer wg.Done()

	t.Logf("\n\nServer: %s", addr.String())

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		t.Logf("connection to server failed: %v", err)
		return
	}
	defer conn.Close()

	resp, err := FetchToken(conn, 5*time.Second)
	if err != nil {
		t.Log(err)
		return
	}

	token, err := ParseToken(resp)

	if err != nil {
		t.Error(err)
		return
	}

	resp, err = FetchWithToken("serverinfo", token, conn, 10*time.Second)
	if err != nil {
		t.Log(err)
		return
	}

	info, err := ParseServerInfo(resp, addr.String())
	if err != nil {
		if errors.Is(err, ErrMalformedResponseData) {
			t.Log(err)
		} else {
			t.Error(err)
		}
		return
	}

	t.Logf("%s\n\n", info.String())
	cnt.Inc()

}

func TestFetchWithTokenServerListAndInfo(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "master1.teeworlds.com:8283")
	if err != nil {
		t.Fatal(err)
	}

	conn, err := net.DialUDP("udp", nil, addr)

	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	var resp []byte
	resp, err = FetchToken(conn, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	var token Token
	token, err = ParseToken(resp)
	if err != nil {
		t.Fatal(err)
	}

	resp, err = FetchWithToken("serverlist", token, conn, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	serverList, err := ParseServerList(resp)
	if err != nil {
		t.Fatal(err)
	}

	var cnt asyncCounter
	wg := sync.WaitGroup{}
	wg.Add(len(serverList))

	for _, s := range serverList {
		s := s
		go getServerInfo(s, t, &wg, &cnt)
	}

	wg.Wait()
	t.Logf("Server Infos retrieved: %s/%d", cnt.String(), len(serverList))
}

func TestFetchWithTokenServerCount(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "master4.teeworlds.com:8283")
	if err != nil {
		t.Fatal(err)
	}

	conn, err := net.DialUDP("udp", nil, addr)

	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	var resp []byte
	resp, err = FetchToken(conn, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	var token Token
	token, err = ParseToken(resp)
	if err != nil {
		t.Fatal(err)
	}

	resp, err = FetchWithToken("servercount", token, conn, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	serverCount, err := ParseServerCount(resp)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Servers fetched: %d", serverCount)
}

func TestServerInfos(t *testing.T) {

	infos := ServerInfos()

	if len(infos) == 0 {
		t.Fatal("expected server list")
	}
}

func BenchmarkServerInfos(b *testing.B) {
	ServerInfos()

}

func TestMatchResponse(t *testing.T) {
	type args struct {
		responseMessage []byte
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"invalid string", args{[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9}}, "", true},
		{"too short payload", args{[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MatchResponse(tt.args.responseMessage)
			if (err != nil) != tt.wantErr {
				t.Errorf("MatchResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("MatchResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}
