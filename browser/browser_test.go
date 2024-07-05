package browser_test

import (
	"testing"
	"time"

	"github.com/jxsl13/twapi/browser"
	"github.com/jxsl13/twapi/internal/testutils/require"
)

func TestGetServerAddresses(t *testing.T) {
	t.Parallel()

	start := time.Now()
	u, err := browser.GetServerAddresses()
	diff := time.Since(start)
	require.NoError(t, err)
	require.NotZero(t, len(u), "found %d server addresses in %d milliseconds", len(u), diff.Milliseconds())
}

func TestGetServerInfos(t *testing.T) {
	t.Parallel()

	start := time.Now()
	u, err := browser.GetServerInfos()
	diff := time.Since(start)
	require.NoError(t, err)
	t.Logf("found %d server infos in %d milliseconds", len(u), diff.Milliseconds())
}

func TestServerInfoOfSingleServer(t *testing.T) {
	t.Parallel()

	start := time.Now()
	u, err := browser.GetServerInfosOf(SimplyzCatch)
	diff := time.Since(start)
	require.NoError(t, err)
	require.Len(t, 1, u)

	t.Logf("found %d server infos in %d milliseconds", len(u), diff.Milliseconds())
}
