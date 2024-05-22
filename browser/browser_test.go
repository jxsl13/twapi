package browser_test

import (
	"testing"
	"time"

	"github.com/jxsl13/twapi/browser"
	"github.com/stretchr/testify/require"
)

func TestGetServerAddresses(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	start := time.Now()
	u, err := browser.GetServerInfos()
	diff := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("found %d server infos in %d milliseconds", len(u), diff.Milliseconds())
}

func TestServerInfoOfSingleServer(t *testing.T) {
	t.Parallel()

	start := time.Now()
	u, err := browser.GetServerInfosOf(SimplyzCatch)
	diff := time.Since(start)
	require.NoError(t, err)
	require.Len(t, u, 1)

	t.Logf("found %d server infos in %d milliseconds", len(u), diff.Milliseconds())
}
