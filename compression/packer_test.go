package compression

import (
	"math/rand"
	"testing"
	"time"

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

	var (
		randoNumbers          = 100_000_000
		randomNumberGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))
		numbers               = make([]int, randoNumbers)
	)

	// generate random numbers
	sign := 0
	for idx := range numbers {

		if idx%2 == 0 {
			sign = -1
		} else {
			sign = 1
		}

		value := sign * int(randomNumberGenerator.Int31())
		numbers[idx] = value
		p.AddInt(value)
	}
	b = p.Bytes()
	u.Reset(b)

	require.Equal(u.Size(), len(b))

	for _, number := range numbers {
		n, err := u.NextInt()
		require.NoError(err)
		require.Equal(n, number)
	}

}
