package compression

import (
	"bytes"
	"io"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func varIntWriteRead(t *testing.T, inNumber int, expectedBytes int) {
	require := require.New(t)

	buf := make([]byte, MaxVarintLen32)
	written := PutVarint(buf, inNumber)
	require.Equal(expectedBytes, written)
	out, read := Varint(buf)
	require.GreaterOrEqual(read, 1, "read must be at least 0")
	require.Equal(inNumber, out, "out == in")
	require.Equal(written, read, "read == written")
	// buf := buf[:written]
}

func TestVarintBoundaries(t *testing.T) {
	t.Parallel()

	varIntWriteRead(t, 63, 1)
	varIntWriteRead(t, 64, 2)

	varIntWriteRead(t, 1048576-1, 3)   // 2^(6+7+7) -1
	varIntWriteRead(t, 1048576, 4)     // 2^(6+7+7)
	varIntWriteRead(t, 134217728-1, 4) // 2^(6+7+7+7) -1
	varIntWriteRead(t, 134217728, 5)   // 2^(6+7+7+7)

	// int32 boundaries
	varIntWriteRead(t, math.MaxInt32, 5) // 2^31 -1 = 2147483647
	varIntWriteRead(t, math.MinInt32, 5) // -2^31 = -2147483648

}

func TestVarintExtensive(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	for in := -20_000_000; in < 20_000_000; in++ {
		var (
			arr       = [MaxVarintLen32]byte{}
			buf       = arr[:]
			written   = PutVarint(buf, in)
			out, read = Varint(buf)
		)
		require.GreaterOrEqual(read, 1, "read must be at least 1")
		require.Equal(in, out, "in/out")
		require.Equal(written, read, "written/read")
	}
}

func TestOverflowVarint(t *testing.T) {
	require := require.New(t)

	buf := []byte{0b10000001, 0b10000001, 0b10000001, 0b10000001, 0b10000001}

	b := bytes.NewBuffer(buf)
	_, err := ReadVarint(b)
	require.Error(err)
}

func TestEOFVarint(t *testing.T) {
	require := require.New(t)

	buf := []byte{0b10000001, 0b10000001, 0b10000001, 0b00000001}
	b := bytes.NewBuffer(buf)

	i, err := ReadVarint(b)
	require.NoError(err)
	require.NotZero(i)

	_, err = ReadVarint(b)
	require.ErrorIs(err, io.EOF)
}
