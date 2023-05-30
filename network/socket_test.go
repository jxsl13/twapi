package network

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewNetSocket(t *testing.T) {
	ns, err := NewNetSocketFrom("127.0.0.1:0", true)
	require.NoError(t, err)
	require.NoError(t, ns.Close())
}
