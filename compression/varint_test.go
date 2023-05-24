package compression

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func varIntWriteRead(t *testing.T, inNumber int, expectedBytes int) {
	require := require.New(t)

	buf := make([]byte, MaxVarintLen32)
	written := PutVarint(buf, inNumber)
	require.Equal(expectedBytes, written)
	out, read := Varint(buf)
	require.Equal(inNumber, out, "out == in")
	require.Equal(written, read, "read == written")
	// buf := buf[:written]
}

func TestVarint(t *testing.T) {

	varIntWriteRead(t, 63, 1)
	varIntWriteRead(t, 64, 2)

	varIntWriteRead(t, 1048576-1, 3)   // 2^(6+7+7) -1
	varIntWriteRead(t, 1048576, 4)     // 2^(6+7+7)
	varIntWriteRead(t, 134217728-1, 4) // 2^(6+7+7+7) -1
	varIntWriteRead(t, 134217728, 5)   // 2^(6+7+7+7)

	require := require.New(t)
	for in := -20_000_000; in < 20_000_000; in++ {
		var (
			arr       = [MaxVarintLen32]byte{}
			buf       = arr[:]
			written   = PutVarint(buf, in)
			out, read = Varint(buf)
		)
		require.Equal(in, out, "in/out")
		require.Equal(written, read, "written/read")
		require.GreaterOrEqual(read, 0, "read must be at least 0")
	}
}

func TestReadVarint(t *testing.T) {
	require := require.New(t)

	buf := []byte{}

	for in := -2_000_000; in < 2_000_000; in++ {
		buf = AppendVarint(buf, in)
	}

	b := bytes.NewBuffer(buf)
	for in := -2_000_000; in < 2_000_000; in++ {
		out, err := ReadVarint(b)
		require.NoError(err)
		require.Equal(in, out, "out != in")
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
