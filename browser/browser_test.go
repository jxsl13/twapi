package browser_test

import (
	"testing"
	"time"

	"github.com/jxsl13/twapi/browser"
)

func TestGetServerAddresses(t *testing.T) {
	start := time.Now()
	u, err := browser.GetServerAddresses()
	diff := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if len(u) == 0 {
		t.Errorf("found %d server addresses in %d milliseconds", len(u), diff.Milliseconds())
	}
}

func TestGetServerInfos(t *testing.T) {
	start := time.Now()
	u, err := browser.GetServerInfos()
	diff := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("found %d server infos in %d milliseconds", len(u), diff.Milliseconds())
}
