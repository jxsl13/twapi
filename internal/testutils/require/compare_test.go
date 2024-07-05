package require

import "testing"

func TestCompare(t *testing.T) {
	GreaterOrEqual(t, 1, 2)
	Greater(t, 1, 2)
	LessOrEqual(t, 2, 1)
	Less(t, 2, 1)

	require := New(t)
	require.GreaterOrEqual(1, 2)
	require.Greater(1, 2)
	require.LessOrEqual(2, 1)
	require.Less(2, 1)
}
