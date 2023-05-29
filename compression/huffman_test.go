package compression

import (
	"encoding/binary"
	"testing"

	"github.com/jxsl13/twapi/protocol"
	"github.com/stretchr/testify/require"
)

func FuzzNewHuffman(f *testing.F) {

	f.Add(toByte(protocol.FrequencyTable[:]))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < protocol.HuffmanMaxSymbols*4 {
			return
		}
		defer func() {
			require.Nil(t, recover())
		}()

		ftSlice := toUint32(data[:protocol.HuffmanMaxSymbols*4])
		var ft [protocol.HuffmanMaxSymbols]uint32
		copy(ft[:], ftSlice)

		h, err := NewHuffman(ft)
		if err != nil {
			return
		}
		require.NotNil(t, h)
	})
}

/*
func FuzzHuffmanCompressDecompress(f *testing.F) {
	h, _ := NewHuffman(protocol.FrequencyTable)

	f.Add([]byte("test"))
	f.Add([]byte("second test"))

	for i := 0; i < 1000; i++ {
		buf := make([]byte, 1500)
		io.ReadFull(rand.Reader, buf)
		f.Add(buf)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < protocol.HuffmanMaxSymbols*4 {
			return
		}
		defer func() {
			require.Nil(t, recover())
		}()
		result := make([]byte, len(data))
		h.Compress()

		ftSlice := toUint32(data[:protocol.HuffmanMaxSymbols*4])
		var ft [protocol.HuffmanMaxSymbols]uint32
		copy(ft[:], ftSlice)

		h, err := NewHuffman(ft)
		if err != nil {
			return
		}
		require.NotNil(t, h)
	})
}
*/

func TestNewHuffman(t *testing.T) {
	h, err := NewHuffman(protocol.FrequencyTable)
	require.NoError(t, err)
	require.NotNil(t, h)
}

func TestHuffmanCompressDecompress(t *testing.T) {
	h, err := NewHuffman(protocol.FrequencyTable)
	require.NoError(t, err)
	require.NotNil(t, h)

	src := []byte("compression test string 01")
	compressed := make([]byte, 1500)
	n, err := h.Compress(src, compressed)
	require.NoError(t, err)
	compressed = compressed[:n]

	decompressed := make([]byte, 1500)
	n, err = h.Decompress(compressed, decompressed)
	require.NoError(t, err)
	decompressed = decompressed[:n]
	require.Equal(t, src, decompressed)
}

func TestUintToBytes(t *testing.T) {
	from := protocol.FrequencyTable[:]
	result := toUint32(toByte(from[:]))

	require.Equal(t, from, result)
}

func toUint32(data []byte) []uint32 {
	resultSize := len(data) / 4
	if resultSize == 0 {
		return []uint32{}
	}
	size := resultSize * 4 // make size a divisor of 4
	data = data[:size]
	result := make([]uint32, resultSize)

	var i int
	for idx := range result {
		i = idx * 4
		result[idx] = binary.BigEndian.Uint32(data[i : i+4])
	}

	return result
}

func toByte(data []uint32) []byte {
	result := make([]byte, len(data)*4)
	var i int
	for idx, ui := range data {
		i = idx * 4
		binary.BigEndian.PutUint32(result[i:i+4], ui)
	}
	return result
}
