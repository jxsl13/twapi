package browser_test

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/jxsl13/twapi/browser"
)

var (
	SimplyzCatch = "89.163.148.121:8305"
	MasterServer = "master1.teeworlds.com:8283"
)

func init() {
	browser.Logging = true
}

func TestClient_GetToken(t *testing.T) {
	c, err := browser.NewClient(MasterServer)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	start := time.Now()
	token, err := c.GetToken()
	if err != nil {
		t.Fatal(err)
	}
	diff := time.Since(start)
	t.Logf("First fetching: %d millis", diff.Milliseconds())

	b, err := json.MarshalIndent(token, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Token: %s", string(b))

	start = time.Now()
	token, err = c.GetToken()
	if err != nil {
		t.Fatal(err)
	}
	diff = time.Since(start)
	t.Logf("Second fetching: %d millis", diff.Microseconds())
	b, err = json.MarshalIndent(token, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Token2: %s", string(b))

}

func TestClient_GetServerCount(t *testing.T) {
	c, err := browser.NewClient(MasterServer)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	i, err := c.GetServerCount()
	if err != nil {
		t.Fatal(err)
	}

	if i <= 0 {
		t.Fatal("<= 0 server count from master servers")
	} else {
		t.Logf("%s has %d registered servers", MasterServer, i)
	}

}

func TestClient_GetServerAddresses(t *testing.T) {
	c, err := browser.NewClient(MasterServer)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	list, err := c.GetServerAddresses()
	if err != nil {
		t.Fatal(err)
	}

	set := map[string]*net.UDPAddr{}
	for _, addr := range list {
		old, ok := set[addr.String()]
		if ok {
			t.Errorf("Duplicate server old: %v new: %v", old.IP, addr.IP)
		}
		set[addr.String()] = addr
	}

	if len(set) != len(list) {
		t.Fatalf("expected unique servers %d, unique servers %d", len(list), len(set))
	}
}

func TestClient_GetServerInfo(t *testing.T) {
	c, err := browser.NewClient(SimplyzCatch)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	si, err := c.GetServerInfo()
	if err != nil {
		t.Fatal(err)
	}

	b, err := json.MarshalIndent(si, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ServerInfo: %s", string(b))
}
