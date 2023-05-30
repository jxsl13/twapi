package compression

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPackerAndUnpacker(t *testing.T) {
	require := require.New(t)

	invalidPacker := NewPacker()

	invalidPacker.AddString("5")
	invalidPacker.AddInt(5)

	invalidUnpacker := NewUnpacker(invalidPacker.Bytes())

	five, err := invalidUnpacker.NextString()
	require.NoError(err)
	require.Equal("5", five)

	_, err = invalidUnpacker.NextString()
	require.ErrorIs(err, ErrNotAString)

	intTest := 5
	stringTest := "abcdefg"
	bytesTest := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}

	p := NewPacker()
	p.AddInt(intTest)
	p.AddString(stringTest)
	p.AddBytes(bytesTest)

	u := NewUnpacker(p.Bytes())

	i, err := u.NextInt()
	require.NoError(err)

	require.Equal(intTest, i)

	s, err := u.NextString()
	require.NoError(err)
	require.Equal(stringTest, s)

	b, err := u.NextBytes(len(bytesTest))
	require.NoError(err)
	require.Equal(bytesTest, b)

	p.Reset()
	require.Zero(p.Size())

	for i := -1_000_000; i < 1_000_000; i++ {
		p.AddInt(i)
	}

	u.Reset(p.Bytes())

	for i := -1_000_000; i < 1_000_000; i++ {
		n, err := u.NextInt()
		require.NoError(err)
		require.Equal(i, n)
	}

}
