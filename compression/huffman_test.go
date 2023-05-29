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

		ftSlice := toInt(data[:protocol.HuffmanMaxSymbols*4])
		var ft [protocol.HuffmanMaxSymbols]int
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
		var ft [protocol.HuffmanMaxSymbols]int
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
	compressed := h.Compress(src)
	require.NoError(t, err)

	decompressed, err := h.Decompress(compressed)
	require.NoError(t, err)

	require.Equal(t, src, decompressed)
}

func TestUintToBytes(t *testing.T) {
	from := protocol.FrequencyTable[:]
	result := toInt(toByte(from[:]))

	require.Equal(t, from, result)
}

func toInt(data []byte) []int {
	resultSize := len(data) / 4
	if resultSize == 0 {
		return []int{}
	}
	size := resultSize * 4 // make size a divisor of 4
	data = data[:size]
	result := make([]int, resultSize)

	var i int
	for idx := range result {
		i = idx * 4
		result[idx] = int(binary.BigEndian.Uint32(data[i : i+4]))
	}

	return result
}

func toByte(data []int) []byte {
	result := make([]byte, len(data)*4)
	var i int
	for idx, ui := range data {
		i = idx * 4
		binary.BigEndian.PutUint32(result[i:i+4], uint32(ui))
	}
	return result
}
