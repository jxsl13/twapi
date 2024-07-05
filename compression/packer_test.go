package compression

import (
	"runtime"
	"sync"
	"testing"

	"github.com/jxsl13/twapi/internal/testutils/require"
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

// rest

func TestUnpackRest(t *testing.T) {
	u := NewUnpacker([]byte{0x01, 0xff, 0xaa})

	{
		got, err := u.NextInt()
		require.NoError(t, err)
		require.Equal(t, 1, got)
	}

	{
		want := []byte{0xff, 0xaa}
		got := u.Bytes()
		require.Equal(t, want, got)
	}
}

func TestUnpackClientInfo(t *testing.T) {
	require := require.New(t)
	u := NewUnpacker([]byte{
		0x24, 0x00, 0x01, 0x00, 0x67, 0x6f, 0x70, 0x68, 0x65, 0x72, 0x00,
		0x00, 0x40, 0x67, 0x72, 0x65, 0x65, 0x6e, 0x73, 0x77, 0x61, 0x72,
		0x64, 0x00, 0x64, 0x75, 0x6f, 0x64, 0x6f, 0x6e, 0x6e, 0x79, 0x00,
		0x00, 0x73, 0x74, 0x61, 0x6e, 0x64, 0x61, 0x72, 0x64, 0x00, 0x73,
		0x74, 0x61, 0x6e, 0x64, 0x61, 0x72, 0x64, 0x00, 0x73, 0x74, 0x61,
		0x6e, 0x64, 0x61, 0x72, 0x64, 0x00, 0x01, 0x01, 0x00, 0x00, 0x00,
		0x00, 0x80, 0xfc, 0xaf, 0x05, 0xeb, 0x83, 0xd0, 0x0a, 0x80, 0xfe,
		0x07, 0x80, 0xfe, 0x07, 0x80, 0xfe, 0x07, 0x80, 0xfe, 0x07, 0x00,
	})

	{
		// message id
		want := 36
		got, err := u.NextInt()
		require.NoError(err)
		require.Equal(want, got)

		// client id
		want = 0
		got, err = u.NextInt()
		require.NoError(err)
		require.Equal(want, got)

		_, err = u.NextBool() // Local bool
		require.NoError(err)
		_, err = u.NextInt() // Team int
		require.NoError(err)
	}

	{
		// name
		want := "gopher"
		got, err := u.NextString()
		require.NoError(err)
		require.Equal(want, got)

		// clan
		want = ""
		got, err = u.NextString()
		require.NoError(err)
		require.Equal(want, got)

	}

	{
		// country
		want := -1
		got, err := u.NextInt()
		require.NoError(err)
		require.Equal(want, got)
	}

	{
		// body
		want := "greensward"
		got, err := u.NextString()
		require.NoError(err)
		require.Equal(want, got)
	}
}

// unpack with state

func TestUnpackSimpleInts(t *testing.T) {
	require := require.New(t)
	u := NewUnpacker([]byte{0x01, 0x02, 0x03, 0x0f})

	want := 1
	got, err := u.NextInt()
	require.NoError(err)
	require.Equal(want, got)

	want = 2
	got, err = u.NextInt()
	require.NoError(err)
	require.Equal(want, got)

	want = 3
	got, err = u.NextInt()
	require.NoError(err)
	require.Equal(want, got)

	want = 15
	got, err = u.NextInt()
	require.NoError(err)
	require.Equal(want, got)
}

func TestUnpackString(t *testing.T) {
	require := require.New(t)
	u := NewUnpacker([]byte{'f', 'o', 'o', 0x00})

	want := "foo"
	got, err := u.NextString()
	require.NoError(err)
	require.Equal(want, got)
}

func TestUnpackTwoStrings(t *testing.T) {
	require := require.New(t)
	u := NewUnpacker([]byte{'f', 'o', 'o', 0x00, 'b', 'a', 'r', 0x00})

	want := "foo"
	got, err := u.NextString()
	require.NoError(err)
	require.Equal(want, got)

	want = "bar"
	got, err = u.NextString()
	require.NoError(err)
	require.Equal(want, got)
}

func TestUnpackMixed(t *testing.T) {
	require := require.New(t)
	u := NewUnpacker([]byte{0x0F, 0x0F, 'f', 'o', 'o', 0x00, 'b', 'a', 'r', 0x00, 0x01})

	// ints
	{
		want := 15
		got, err := u.NextInt()
		require.NoError(err)
		require.Equal(want, got)

		want = 15
		got, err = u.NextInt()
		require.NoError(err)
		require.Equal(want, got)
	}

	// strings
	{
		want := "foo"
		got, err := u.NextString()
		require.NoError(err)
		require.Equal(want, got)

		want = "bar"
		got, err = u.NextString()
		require.NoError(err)
		require.Equal(want, got)
	}

	// ints
	{
		want := 1
		got, err := u.NextInt()
		require.NoError(err)
		require.Equal(want, got)
	}
}
