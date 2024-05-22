package browser_test

import (
	"encoding/json"
	"net/netip"
	"testing"
	"time"

	"github.com/jxsl13/twapi/browser"
)

var (
	SimplyzCatch = "89.163.148.121:8303"
	MasterServer = "master1.teeworlds.com:8283"
)

func init() {
	browser.Logging = true
}

func TestClientGetToken(t *testing.T) {
	t.Parallel()

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

func TestClientGetServerCount(t *testing.T) {
	t.Parallel()

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

func TestClientGetServerAddresses(t *testing.T) {
	t.Parallel()

	c, err := browser.NewClient(MasterServer)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	list, err := c.GetServerAddresses()
	if err != nil {
		t.Fatal(err)
	}

	set := map[netip.AddrPort]struct{}{}
	for _, addr := range list {
		old, ok := set[addr]
		if ok {
			t.Errorf("Duplicate server old: %v", old)
		}
		set[addr] = struct{}{}
	}

	if len(set) != len(list) {
		t.Fatalf("expected unique servers %d, unique servers %d", len(list), len(set))
	}
}

func TestClientGetServerInfo(t *testing.T) {
	t.Parallel()

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

func TestGetSingleMasterServer(t *testing.T) {
	t.Parallel()

	client, err := browser.NewClient(MasterServer)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	addresses, err := client.GetServerAddresses()
	if err != nil {
		t.Fatal(err)
	}

	t.Log(addresses)
}
