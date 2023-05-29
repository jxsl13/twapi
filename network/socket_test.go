package network

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewNetSocket(t *testing.T) {
	ns, err := NewNetSocketFrom(":", true)
	require.NoError(t, err)
	require.NoError(t, ns.Close())
}
