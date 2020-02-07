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
		t.Error(err)
		return
	}
	defer conn.Close()

	resp, err := RetryFetchToken(conn, 5*time.Second)
	if err != nil {
		if errors.Is(err, ErrTimeout) {
			t.Log(err)
			return
		}

		t.Error(err)
		return
	}

	token, err := ParseToken(resp)

	if err != nil {
		t.Error(err)
		return
	}

	resp, err = RetryFetch("serverinfo", token, conn, 5*time.Second)
	if err != nil {
		t.Error(err)
		return
	}

	info, err := ParseServerInfo(resp, addr.String())
	if err != nil {
		t.Error(err)
		return
	}

	t.Logf("%s\n\n", info.String())
	cnt.Inc()

}

func TestRetryFetch(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "master1.teeworlds.com:8283")
	if err != nil {
		t.Fatal(err)
	}

	conn, err := net.DialUDP("udp", nil, addr)

	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	resp, err := RetryFetchToken(conn, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	token, err := ParseToken(resp)
	if err != nil {
		t.Fatal(err)
	}

	resp, err = RetryFetch("serverlist", token, conn, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	serverList, err := ParseServerList(resp)
	if err != nil {
		t.Fatal(err)
	}

	// fetch serer count
	resp, err = RetryFetchToken(conn, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	token, err = ParseToken(resp)
	if err != nil {
		t.Fatal(err)
	}

	resp, err = RetryFetch("servercount", token, conn, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	serverCount, err := ParseServerCount(resp)
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
	t.Logf("Server Infos retrieved: %s/%d(server count: %d)", cnt.String(), len(serverList), serverCount)
}
