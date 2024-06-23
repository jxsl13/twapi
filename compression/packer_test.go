package compression

import (
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReset(t *testing.T) {
	p := NewPacker([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9})

	expected := []byte{10, 11, 12, 13, 14, 15, 16}
	p.Reset(expected)

	result := p.Bytes()
	require.Equal(t, expected, result)

}

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

	// do batches instead of 20 million numbers in memory
	batch := func(start, end int, wg *sync.WaitGroup) {
		defer wg.Done()
		p := NewPacker(make([]byte, 0, (end-start)*2*4 /* = 4 byte + extra space*/))
		p.Reset()
		for i := start; i < end; i++ {
			p.AddInt(i)
		}

		u := NewUnpacker(p.Bytes())
		u.Reset(p.Bytes())
		for i := start; i < end; i++ {
			ui, err := u.NextInt()
			require.NoError(err)
			require.Equal(i, ui)
		}
	}

	var (
		start     = -20_000_000
		end       = 20_000_000
		batches   = runtime.NumCPU()
		batchSize = (end - start) / batches
	)
	if (end-start)%batchSize > 0 {
		batches += 1
	}

	var wg sync.WaitGroup
	wg.Add(batches)

	for i := start; i < end; i += batchSize {
		batch(i, i+batchSize, &wg)
	}

	wg.Wait()

}
