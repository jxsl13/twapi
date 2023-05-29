package compression

import (
	"crypto/rand"
	"encoding/binary"
	"io"
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
		require.NoError(t, err)
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

func TestHuffmanCompress(t *testing.T) {
	h, err := NewHuffman(protocol.FrequencyTable)
	require.NoError(t, err)
	require.NotNil(t, h)

	src := []byte("compression test string 01")
	compressed := h.Compress(src)

	decompressed := make([]byte, len(src)*10)
	n := HuffmanDecompress(compressed, len(compressed), decompressed, len(decompressed))
	require.Greater(t, n, -1)
	decompressed = decompressed[:n]
	require.NoError(t, err)

	require.Equal(t, src, decompressed)
}

func FuzzHuffmanCompress(f *testing.F) {
	h, _ := NewHuffman(protocol.FrequencyTable)

	buf := [1500]byte{}

	for i := 0; i < 100; i++ {
		_, err := io.ReadFull(rand.Reader, buf[:])
		if err != nil {
			panic(err)
		}
		f.Add(buf[:])
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		compressed := h.Compress(data)

		result := make([]byte, 0, len(data)*2)
		n := HuffmanDecompress(compressed, len(compressed), result, len(result))
		require.GreaterOrEqual(t, n, 0)
		result = result[:n]

		require.Equal(t, data, result)
	})
}

func TestHuffmanCompressDecompress(t *testing.T) {
	h, err := NewHuffman(protocol.FrequencyTable)
	require.NoError(t, err)
	require.NotNil(t, h)

	src := []byte("compression test string 01")
	compressed := h.Compress(src)
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

type HuffmanNode struct {
	Bits    uint32
	NumBits uint32
	ALeafs  [2]uint16
	Symbol  byte
}

func HuffmanDecompress(inBuffer []byte, inBufferSize int, outBuffer []byte, outBufferSize int) int {

	pCurrentByte := 0
	pCurrentBit := byte(0)
	pCurrentNode := uint16(len(HuffmanTree) - 1)
	pOut := 0

	for {
		if pCurrentByte > inBufferSize-1 {
			return -1
		}
		bit := (inBuffer[pCurrentByte] & (1 << pCurrentBit)) >> pCurrentBit

		pCurrentNode = HuffmanTree[pCurrentNode].ALeafs[bit]

		if pCurrentNode == HuffmanEOFSymbol {
			return pOut // return out size
		}

		// if symbol was hit
		if HuffmanTree[pCurrentNode].NumBits != 0 {

			if pOut > outBufferSize-1 {
				return -1
			}

			outBuffer[pOut] = HuffmanTree[pCurrentNode].Symbol
			pOut += 1
			pCurrentNode = uint16(len(HuffmanTree) - 1)
		}

		pCurrentBit += 1
		if pCurrentBit == 8 {
			pCurrentBit = 0
			pCurrentByte += 1
		}
	}
}

var HuffmanTree = [HuffmanMaxNodes]HuffmanNode{
	{1, 1, [2]uint16{65535, 65535}, 0},
	{8, 4, [2]uint16{65535, 65535}, 1},
	{2, 5, [2]uint16{65535, 65535}, 2},
	{22, 8, [2]uint16{65535, 65535}, 3},
	{30, 6, [2]uint16{65535, 65535}, 4},
	{118, 7, [2]uint16{65535, 65535}, 5},
	{54, 8, [2]uint16{65535, 65535}, 6},
	{110, 8, [2]uint16{65535, 65535}, 7},
	{4, 5, [2]uint16{65535, 65535}, 8},
	{76, 7, [2]uint16{65535, 65535}, 9},
	{122, 7, [2]uint16{65535, 65535}, 10},
	{254, 8, [2]uint16{65535, 65535}, 11},
	{114, 7, [2]uint16{65535, 65535}, 12},
	{14, 6, [2]uint16{65535, 65535}, 13},
	{244, 8, [2]uint16{65535, 65535}, 14},
	{238, 9, [2]uint16{65535, 65535}, 15},
	{106, 7, [2]uint16{65535, 65535}, 16},
	{166, 9, [2]uint16{65535, 65535}, 17},
	{92, 7, [2]uint16{65535, 65535}, 18},
	{170, 9, [2]uint16{65535, 65535}, 19},
	{240, 8, [2]uint16{65535, 65535}, 20},
	{446, 10, [2]uint16{65535, 65535}, 21},
	{52, 9, [2]uint16{65535, 65535}, 22},
	{124, 9, [2]uint16{65535, 65535}, 23},
	{16, 7, [2]uint16{65535, 65535}, 24},
	{316, 9, [2]uint16{65535, 65535}, 25},
	{18, 9, [2]uint16{65535, 65535}, 26},
	{208, 9, [2]uint16{65535, 65535}, 27},
	{10, 9, [2]uint16{65535, 65535}, 28},
	{102, 7, [2]uint16{65535, 65535}, 29},
	{202, 8, [2]uint16{65535, 65535}, 30},
	{302, 10, [2]uint16{65535, 65535}, 31},
	{428, 9, [2]uint16{65535, 65535}, 32},
	{140, 9, [2]uint16{65535, 65535}, 33},
	{442, 9, [2]uint16{65535, 65535}, 34},
	{934, 10, [2]uint16{65535, 65535}, 35},
	{48, 9, [2]uint16{65535, 65535}, 36},
	{138, 9, [2]uint16{65535, 65535}, 37},
	{474, 9, [2]uint16{65535, 65535}, 38},
	{178, 9, [2]uint16{65535, 65535}, 39},
	{86, 7, [2]uint16{65535, 65535}, 40},
	{58, 8, [2]uint16{65535, 65535}, 41},
	{274, 9, [2]uint16{65535, 65535}, 42},
	{382, 10, [2]uint16{65535, 65535}, 43},
	{436, 9, [2]uint16{65535, 65535}, 44},
	{694, 10, [2]uint16{65535, 65535}, 45},
	{558, 10, [2]uint16{65535, 65535}, 46},
	{406, 10, [2]uint16{65535, 65535}, 47},
	{942, 10, [2]uint16{65535, 65535}, 48},
	{318, 10, [2]uint16{65535, 65535}, 49},
	{190, 10, [2]uint16{65535, 65535}, 50},
	{858, 10, [2]uint16{65535, 65535}, 51},
	{572, 11, [2]uint16{65535, 65535}, 52},
	{3356, 12, [2]uint16{65535, 65535}, 53},
	{1726, 13, [2]uint16{65535, 65535}, 54},
	{906, 12, [2]uint16{65535, 65535}, 55},
	{1492, 12, [2]uint16{65535, 65535}, 56},
	{3696, 12, [2]uint16{65535, 65535}, 57},
	{2640, 12, [2]uint16{65535, 65535}, 58},
	{1652, 12, [2]uint16{65535, 65535}, 59},
	{7230, 13, [2]uint16{65535, 65535}, 60},
	{116, 11, [2]uint16{65535, 65535}, 61},
	{1446, 13, [2]uint16{65535, 65535}, 62},
	{338, 12, [2]uint16{65535, 65535}, 63},
	{74, 8, [2]uint16{65535, 65535}, 64},
	{444, 9, [2]uint16{65535, 65535}, 65},
	{346, 10, [2]uint16{65535, 65535}, 66},
	{908, 10, [2]uint16{65535, 65535}, 67},
	{410, 10, [2]uint16{65535, 65535}, 68},
	{434, 10, [2]uint16{65535, 65535}, 69},
	{60, 10, [2]uint16{65535, 65535}, 70},
	{112, 10, [2]uint16{65535, 65535}, 71},
	{380, 10, [2]uint16{65535, 65535}, 72},
	{148, 8, [2]uint16{65535, 65535}, 73},
	{298, 10, [2]uint16{65535, 65535}, 74},
	{426, 9, [2]uint16{65535, 65535}, 75},
	{780, 10, [2]uint16{65535, 65535}, 76},
	{464, 10, [2]uint16{65535, 65535}, 77},
	{1062, 11, [2]uint16{65535, 65535}, 78},
	{1918, 11, [2]uint16{65535, 65535}, 79},
	{174, 11, [2]uint16{65535, 65535}, 80},
	{1982, 11, [2]uint16{65535, 65535}, 81},
	{62, 11, [2]uint16{65535, 65535}, 82},
	{108, 7, [2]uint16{65535, 65535}, 83},
	{532, 10, [2]uint16{65535, 65535}, 84},
	{172, 9, [2]uint16{65535, 65535}, 85},
	{186, 11, [2]uint16{65535, 65535}, 86},
	{26, 11, [2]uint16{65535, 65535}, 87},
	{816, 10, [2]uint16{65535, 65535}, 88},
	{1362, 11, [2]uint16{65535, 65535}, 89},
	{2954, 13, [2]uint16{65535, 65535}, 90},
	{1946, 12, [2]uint16{65535, 65535}, 91},
	{7354, 13, [2]uint16{65535, 65535}, 92},
	{3188, 12, [2]uint16{65535, 65535}, 93},
	{1916, 13, [2]uint16{65535, 65535}, 94},
	{5018, 13, [2]uint16{65535, 65535}, 95},
	{922, 13, [2]uint16{65535, 65535}, 96},
	{1308, 12, [2]uint16{65535, 65535}, 97},
	{6326, 13, [2]uint16{65535, 65535}, 98},
	{4564, 13, [2]uint16{65535, 65535}, 99},
	{2204, 12, [2]uint16{65535, 65535}, 100},
	{156, 12, [2]uint16{65535, 65535}, 101},
	{7796, 13, [2]uint16{65535, 65535}, 102},
	{3700, 13, [2]uint16{65535, 65535}, 103},
	{5550, 13, [2]uint16{65535, 65535}, 104},
	{1454, 13, [2]uint16{65535, 65535}, 105},
	{7066, 13, [2]uint16{65535, 65535}, 106},
	{2970, 13, [2]uint16{65535, 65535}, 107},
	{1198, 14, [2]uint16{65535, 65535}, 108},
	{1140, 12, [2]uint16{65535, 65535}, 109},
	{7598, 13, [2]uint16{65535, 65535}, 110},
	{2230, 13, [2]uint16{65535, 65535}, 111},
	{3258, 13, [2]uint16{65535, 65535}, 112},
	{1648, 12, [2]uint16{65535, 65535}, 113},
	{624, 11, [2]uint16{65535, 65535}, 114},
	{2676, 12, [2]uint16{65535, 65535}, 115},
	{592, 12, [2]uint16{65535, 65535}, 116},
	{5302, 13, [2]uint16{65535, 65535}, 117},
	{5822, 14, [2]uint16{65535, 65535}, 118},
	{23434, 15, [2]uint16{65535, 65535}, 119},
	{3502, 13, [2]uint16{65535, 65535}, 120},
	{3664, 12, [2]uint16{65535, 65535}, 121},
	{468, 13, [2]uint16{65535, 65535}, 122},
	{4526, 13, [2]uint16{65535, 65535}, 123},
	{1206, 13, [2]uint16{65535, 65535}, 124},
	{2900, 12, [2]uint16{65535, 65535}, 125},
	{9390, 14, [2]uint16{65535, 65535}, 126},
	{5916, 13, [2]uint16{65535, 65535}, 127},
	{0, 5, [2]uint16{65535, 65535}, 128},
	{638, 10, [2]uint16{65535, 65535}, 129},
	{300, 9, [2]uint16{65535, 65535}, 130},
	{894, 11, [2]uint16{65535, 65535}, 131},
	{282, 9, [2]uint16{65535, 65535}, 132},
	{494, 11, [2]uint16{65535, 65535}, 133},
	{82, 9, [2]uint16{65535, 65535}, 134},
	{1710, 11, [2]uint16{65535, 65535}, 135},
	{90, 9, [2]uint16{65535, 65535}, 136},
	{574, 11, [2]uint16{65535, 65535}, 137},
	{306, 9, [2]uint16{65535, 65535}, 138},
	{20, 10, [2]uint16{65535, 65535}, 139},
	{180, 9, [2]uint16{65535, 65535}, 140},
	{1150, 11, [2]uint16{65535, 65535}, 141},
	{508, 9, [2]uint16{65535, 65535}, 142},
	{396, 10, [2]uint16{65535, 65535}, 143},
	{294, 9, [2]uint16{65535, 65535}, 144},
	{126, 11, [2]uint16{65535, 65535}, 145},
	{44, 9, [2]uint16{65535, 65535}, 146},
	{304, 10, [2]uint16{65535, 65535}, 147},
	{6, 6, [2]uint16{65535, 65535}, 148},
	{394, 10, [2]uint16{65535, 65535}, 149},
	{50, 9, [2]uint16{65535, 65535}, 150},
	{1574, 11, [2]uint16{65535, 65535}, 151},
	{308, 9, [2]uint16{65535, 65535}, 152},
	{946, 10, [2]uint16{65535, 65535}, 153},
	{42, 9, [2]uint16{65535, 65535}, 154},
	{698, 11, [2]uint16{65535, 65535}, 155},
	{466, 9, [2]uint16{65535, 65535}, 156},
	{1598, 11, [2]uint16{65535, 65535}, 157},
	{412, 9, [2]uint16{65535, 65535}, 158},
	{668, 10, [2]uint16{65535, 65535}, 159},
	{252, 9, [2]uint16{65535, 65535}, 160},
	{538, 10, [2]uint16{65535, 65535}, 161},
	{830, 10, [2]uint16{65535, 65535}, 162},
	{1006, 10, [2]uint16{65535, 65535}, 163},
	{438, 9, [2]uint16{65535, 65535}, 164},
	{12, 9, [2]uint16{65535, 65535}, 165},
	{150, 9, [2]uint16{65535, 65535}, 166},
	{268, 10, [2]uint16{65535, 65535}, 167},
	{218, 9, [2]uint16{65535, 65535}, 168},
	{918, 10, [2]uint16{65535, 65535}, 169},
	{154, 9, [2]uint16{65535, 65535}, 170},
	{980, 10, [2]uint16{65535, 65535}, 171},
	{210, 9, [2]uint16{65535, 65535}, 172},
	{976, 10, [2]uint16{65535, 65535}, 173},
	{212, 9, [2]uint16{65535, 65535}, 174},
	{958, 11, [2]uint16{65535, 65535}, 175},
	{176, 8, [2]uint16{65535, 65535}, 176},
	{84, 9, [2]uint16{65535, 65535}, 177},
	{368, 9, [2]uint16{65535, 65535}, 178},
	{686, 11, [2]uint16{65535, 65535}, 179},
	{336, 9, [2]uint16{65535, 65535}, 180},
	{550, 11, [2]uint16{65535, 65535}, 181},
	{266, 9, [2]uint16{65535, 65535}, 182},
	{1838, 11, [2]uint16{65535, 65535}, 183},
	{276, 9, [2]uint16{65535, 65535}, 184},
	{814, 11, [2]uint16{65535, 65535}, 185},
	{28, 9, [2]uint16{65535, 65535}, 186},
	{1518, 11, [2]uint16{65535, 65535}, 187},
	{188, 9, [2]uint16{65535, 65535}, 188},
	{46, 10, [2]uint16{65535, 65535}, 189},
	{372, 9, [2]uint16{65535, 65535}, 190},
	{810, 10, [2]uint16{65535, 65535}, 191},
	{3770, 12, [2]uint16{65535, 65535}, 192},
	{2898, 12, [2]uint16{65535, 65535}, 193},
	{3644, 12, [2]uint16{65535, 65535}, 194},
	{3254, 12, [2]uint16{65535, 65535}, 195},
	{3228, 12, [2]uint16{65535, 65535}, 196},
	{1722, 12, [2]uint16{65535, 65535}, 197},
	{1086, 12, [2]uint16{65535, 65535}, 198},
	{1596, 12, [2]uint16{65535, 65535}, 199},
	{2750, 12, [2]uint16{65535, 65535}, 200},
	{182, 12, [2]uint16{65535, 65535}, 201},
	{1180, 12, [2]uint16{65535, 65535}, 202},
	{1616, 12, [2]uint16{65535, 65535}, 203},
	{3978, 12, [2]uint16{65535, 65535}, 204},
	{2940, 12, [2]uint16{65535, 65535}, 205},
	{3246, 12, [2]uint16{65535, 65535}, 206},
	{850, 12, [2]uint16{65535, 65535}, 207},
	{3922, 12, [2]uint16{65535, 65535}, 208},
	{892, 12, [2]uint16{65535, 65535}, 209},
	{2470, 12, [2]uint16{65535, 65535}, 210},
	{1210, 12, [2]uint16{65535, 65535}, 211},
	{2128, 12, [2]uint16{65535, 65535}, 212},
	{422, 12, [2]uint16{65535, 65535}, 213},
	{1930, 12, [2]uint16{65535, 65535}, 214},
	{852, 12, [2]uint16{65535, 65535}, 215},
	{284, 11, [2]uint16{65535, 65535}, 216},
	{2086, 12, [2]uint16{65535, 65535}, 217},
	{430, 13, [2]uint16{65535, 65535}, 218},
	{3964, 12, [2]uint16{65535, 65535}, 219},
	{5146, 13, [2]uint16{65535, 65535}, 220},
	{3924, 12, [2]uint16{65535, 65535}, 221},
	{1876, 12, [2]uint16{65535, 65535}, 222},
	{3494, 12, [2]uint16{65535, 65535}, 223},
	{2844, 12, [2]uint16{65535, 65535}, 224},
	{2388, 12, [2]uint16{65535, 65535}, 225},
	{6574, 13, [2]uint16{65535, 65535}, 226},
	{628, 12, [2]uint16{65535, 65535}, 227},
	{6012, 13, [2]uint16{65535, 65535}, 228},
	{80, 12, [2]uint16{65535, 65535}, 229},
	{3774, 12, [2]uint16{65535, 65535}, 230},
	{1050, 13, [2]uint16{65535, 65535}, 231},
	{340, 12, [2]uint16{65535, 65535}, 232},
	{2478, 13, [2]uint16{65535, 65535}, 233},
	{3540, 12, [2]uint16{65535, 65535}, 234},
	{3412, 12, [2]uint16{65535, 65535}, 235},
	{3134, 13, [2]uint16{65535, 65535}, 236},
	{38, 12, [2]uint16{65535, 65535}, 237},
	{3152, 12, [2]uint16{65535, 65535}, 238},
	{1364, 12, [2]uint16{65535, 65535}, 239},
	{14014, 14, [2]uint16{65535, 65535}, 240},
	{1820, 13, [2]uint16{65535, 65535}, 241},
	{3994, 12, [2]uint16{65535, 65535}, 242},
	{1874, 12, [2]uint16{65535, 65535}, 243},
	{2386, 12, [2]uint16{65535, 65535}, 244},
	{1104, 12, [2]uint16{65535, 65535}, 245},
	{796, 12, [2]uint16{65535, 65535}, 246},
	{4798, 13, [2]uint16{65535, 65535}, 247},
	{15242, 14, [2]uint16{65535, 65535}, 248},
	{702, 13, [2]uint16{65535, 65535}, 249},
	{5294, 13, [2]uint16{65535, 65535}, 250},
	{3868, 12, [2]uint16{65535, 65535}, 251},
	{5542, 13, [2]uint16{65535, 65535}, 252},
	{2516, 12, [2]uint16{65535, 65535}, 253},
	{3098, 12, [2]uint16{65535, 65535}, 254},
	{146, 8, [2]uint16{65535, 65535}, 255},
	{7050, 15, [2]uint16{65535, 65535}, 0},
	{0, 0, [2]uint16{256, 119}, 0},
	{0, 0, [2]uint16{257, 248}, 0},
	{0, 0, [2]uint16{108, 126}, 0},
	{0, 0, [2]uint16{118, 240}, 0},
	{0, 0, [2]uint16{122, 99}, 0},
	{0, 0, [2]uint16{241, 127}, 0},
	{0, 0, [2]uint16{103, 102}, 0},
	{0, 0, [2]uint16{94, 228}, 0},
	{0, 0, [2]uint16{90, 258}, 0},
	{0, 0, [2]uint16{231, 220}, 0},
	{0, 0, [2]uint16{107, 106}, 0},
	{0, 0, [2]uint16{96, 95}, 0},
	{0, 0, [2]uint16{112, 92}, 0},
	{0, 0, [2]uint16{62, 252}, 0},
	{0, 0, [2]uint16{124, 117}, 0},
	{0, 0, [2]uint16{111, 98}, 0},
	{0, 0, [2]uint16{259, 250}, 0},
	{0, 0, [2]uint16{233, 226}, 0},
	{0, 0, [2]uint16{218, 123}, 0},
	{0, 0, [2]uint16{120, 110}, 0},
	{0, 0, [2]uint16{105, 104}, 0},
	{0, 0, [2]uint16{249, 247}, 0},
	{0, 0, [2]uint16{236, 60}, 0},
	{0, 0, [2]uint16{54, 260}, 0},
	{0, 0, [2]uint16{245, 238}, 0},
	{0, 0, [2]uint16{229, 212}, 0},
	{0, 0, [2]uint16{203, 121}, 0},
	{0, 0, [2]uint16{116, 58}, 0},
	{0, 0, [2]uint16{261, 253}, 0},
	{0, 0, [2]uint16{239, 235}, 0},
	{0, 0, [2]uint16{232, 225}, 0},
	{0, 0, [2]uint16{222, 221}, 0},
	{0, 0, [2]uint16{215, 125}, 0},
	{0, 0, [2]uint16{113, 57}, 0},
	{0, 0, [2]uint16{56, 234}, 0},
	{0, 0, [2]uint16{227, 115}, 0},
	{0, 0, [2]uint16{109, 93}, 0},
	{0, 0, [2]uint16{59, 263}, 0},
	{0, 0, [2]uint16{262, 251}, 0},
	{0, 0, [2]uint16{246, 224}, 0},
	{0, 0, [2]uint16{202, 196}, 0},
	{0, 0, [2]uint16{101, 100}, 0},
	{0, 0, [2]uint16{97, 53}, 0},
	{0, 0, [2]uint16{264, 219}, 0},
	{0, 0, [2]uint16{209, 205}, 0},
	{0, 0, [2]uint16{199, 194}, 0},
	{0, 0, [2]uint16{63, 244}, 0},
	{0, 0, [2]uint16{243, 208}, 0},
	{0, 0, [2]uint16{207, 193}, 0},
	{0, 0, [2]uint16{55, 265}, 0},
	{0, 0, [2]uint16{214, 204}, 0},
	{0, 0, [2]uint16{268, 267}, 0},
	{0, 0, [2]uint16{266, 254}, 0},
	{0, 0, [2]uint16{91, 242}, 0},
	{0, 0, [2]uint16{211, 269}, 0},
	{0, 0, [2]uint16{237, 217}, 0},
	{0, 0, [2]uint16{197, 192}, 0},
	{0, 0, [2]uint16{270, 223}, 0},
	{0, 0, [2]uint16{213, 210}, 0},
	{0, 0, [2]uint16{201, 272}, 0},
	{0, 0, [2]uint16{271, 195}, 0},
	{0, 0, [2]uint16{277, 276}, 0},
	{0, 0, [2]uint16{275, 274}, 0},
	{0, 0, [2]uint16{273, 206}, 0},
	{0, 0, [2]uint16{198, 279}, 0},
	{0, 0, [2]uint16{278, 200}, 0},
	{0, 0, [2]uint16{280, 230}, 0},
	{0, 0, [2]uint16{284, 283}, 0},
	{0, 0, [2]uint16{282, 281}, 0},
	{0, 0, [2]uint16{114, 290}, 0},
	{0, 0, [2]uint16{289, 288}, 0},
	{0, 0, [2]uint16{287, 286}, 0},
	{0, 0, [2]uint16{285, 291}, 0},
	{0, 0, [2]uint16{61, 293}, 0},
	{0, 0, [2]uint16{292, 294}, 0},
	{0, 0, [2]uint16{216, 299}, 0},
	{0, 0, [2]uint16{298, 297}, 0},
	{0, 0, [2]uint16{296, 295}, 0},
	{0, 0, [2]uint16{52, 302}, 0},
	{0, 0, [2]uint16{301, 300}, 0},
	{0, 0, [2]uint16{303, 89}, 0},
	{0, 0, [2]uint16{305, 304}, 0},
	{0, 0, [2]uint16{306, 307}, 0},
	{0, 0, [2]uint16{87, 309}, 0},
	{0, 0, [2]uint16{308, 310}, 0},
	{0, 0, [2]uint16{86, 311}, 0},
	{0, 0, [2]uint16{155, 313}, 0},
	{0, 0, [2]uint16{312, 78}, 0},
	{0, 0, [2]uint16{181, 151}, 0},
	{0, 0, [2]uint16{315, 314}, 0},
	{0, 0, [2]uint16{316, 317}, 0},
	{0, 0, [2]uint16{185, 183}, 0},
	{0, 0, [2]uint16{80, 320}, 0},
	{0, 0, [2]uint16{319, 318}, 0},
	{0, 0, [2]uint16{179, 135}, 0},
	{0, 0, [2]uint16{133, 187}, 0},
	{0, 0, [2]uint16{82, 321}, 0},
	{0, 0, [2]uint16{137, 157}, 0},
	{0, 0, [2]uint16{322, 323}, 0},
	{0, 0, [2]uint16{175, 81}, 0},
	{0, 0, [2]uint16{145, 141}, 0},
	{0, 0, [2]uint16{325, 324}, 0},
	{0, 0, [2]uint16{131, 79}, 0},
	{0, 0, [2]uint16{77, 173}, 0},
	{0, 0, [2]uint16{147, 88}, 0},
	{0, 0, [2]uint16{71, 326}, 0},
	{0, 0, [2]uint16{139, 84}, 0},
	{0, 0, [2]uint16{328, 327}, 0},
	{0, 0, [2]uint16{329, 171}, 0},
	{0, 0, [2]uint16{330, 331}, 0},
	{0, 0, [2]uint16{167, 76}, 0},
	{0, 0, [2]uint16{143, 67}, 0},
	{0, 0, [2]uint16{332, 334}, 0},
	{0, 0, [2]uint16{333, 159}, 0},
	{0, 0, [2]uint16{70, 335}, 0},
	{0, 0, [2]uint16{72, 336}, 0},
	{0, 0, [2]uint16{337, 338}, 0},
	{0, 0, [2]uint16{69, 153}, 0},
	{0, 0, [2]uint16{149, 339}, 0},
	{0, 0, [2]uint16{74, 191}, 0},
	{0, 0, [2]uint16{340, 161}, 0},
	{0, 0, [2]uint16{68, 341}, 0},
	{0, 0, [2]uint16{66, 51}, 0},
	{0, 0, [2]uint16{342, 343}, 0},
	{0, 0, [2]uint16{344, 345}, 0},
	{0, 0, [2]uint16{346, 35}, 0},
	{0, 0, [2]uint16{47, 169}, 0},
	{0, 0, [2]uint16{347, 45}, 0},
	{0, 0, [2]uint16{189, 46}, 0},
	{0, 0, [2]uint16{31, 348}, 0},
	{0, 0, [2]uint16{349, 351}, 0},
	{0, 0, [2]uint16{350, 48}, 0},
	{0, 0, [2]uint16{352, 163}, 0},
	{0, 0, [2]uint16{353, 354}, 0},
	{0, 0, [2]uint16{49, 162}, 0},
	{0, 0, [2]uint16{50, 355}, 0},
	{0, 0, [2]uint16{21, 356}, 0},
	{0, 0, [2]uint16{357, 129}, 0},
	{0, 0, [2]uint16{43, 359}, 0},
	{0, 0, [2]uint16{358, 180}, 0},
	{0, 0, [2]uint16{27, 360}, 0},
	{0, 0, [2]uint16{36, 361}, 0},
	{0, 0, [2]uint16{362, 178}, 0},
	{0, 0, [2]uint16{363, 184}, 0},
	{0, 0, [2]uint16{177, 364}, 0},
	{0, 0, [2]uint16{174, 365}, 0},
	{0, 0, [2]uint16{22, 152}, 0},
	{0, 0, [2]uint16{140, 44}, 0},
	{0, 0, [2]uint16{366, 190}, 0},
	{0, 0, [2]uint16{165, 367}, 0},
	{0, 0, [2]uint16{33, 368}, 0},
	{0, 0, [2]uint16{146, 130}, 0},
	{0, 0, [2]uint16{85, 32}, 0},
	{0, 0, [2]uint16{186, 369}, 0},
	{0, 0, [2]uint16{370, 158}, 0},
	{0, 0, [2]uint16{371, 25}, 0},
	{0, 0, [2]uint16{188, 65}, 0},
	{0, 0, [2]uint16{23, 372}, 0},
	{0, 0, [2]uint16{160, 142}, 0},
	{0, 0, [2]uint16{26, 42}, 0},
	{0, 0, [2]uint16{134, 373}, 0},
	{0, 0, [2]uint16{172, 156}, 0},
	{0, 0, [2]uint16{150, 138}, 0},
	{0, 0, [2]uint16{39, 374}, 0},
	{0, 0, [2]uint16{28, 182}, 0},
	{0, 0, [2]uint16{37, 375}, 0},
	{0, 0, [2]uint16{154, 376}, 0},
	{0, 0, [2]uint16{19, 75}, 0},
	{0, 0, [2]uint16{377, 132}, 0},
	{0, 0, [2]uint16{170, 378}, 0},
	{0, 0, [2]uint16{136, 379}, 0},
	{0, 0, [2]uint16{168, 38}, 0},
	{0, 0, [2]uint16{380, 34}, 0},
	{0, 0, [2]uint16{381, 144}, 0},
	{0, 0, [2]uint16{17, 382}, 0},
	{0, 0, [2]uint16{166, 383}, 0},
	{0, 0, [2]uint16{384, 164}, 0},
	{0, 0, [2]uint16{385, 386}, 0},
	{0, 0, [2]uint16{387, 388}, 0},
	{0, 0, [2]uint16{15, 389}, 0},
	{0, 0, [2]uint16{390, 391}, 0},
	{0, 0, [2]uint16{392, 393}, 0},
	{0, 0, [2]uint16{394, 395}, 0},
	{0, 0, [2]uint16{396, 397}, 0},
	{0, 0, [2]uint16{398, 176}, 0},
	{0, 0, [2]uint16{399, 20}, 0},
	{0, 0, [2]uint16{400, 73}, 0},
	{0, 0, [2]uint16{401, 402}, 0},
	{0, 0, [2]uint16{403, 404}, 0},
	{0, 0, [2]uint16{405, 14}, 0},
	{0, 0, [2]uint16{406, 407}, 0},
	{0, 0, [2]uint16{408, 409}, 0},
	{0, 0, [2]uint16{410, 411}, 0},
	{0, 0, [2]uint16{412, 413}, 0},
	{0, 0, [2]uint16{414, 415}, 0},
	{0, 0, [2]uint16{416, 255}, 0},
	{0, 0, [2]uint16{417, 418}, 0},
	{0, 0, [2]uint16{419, 420}, 0},
	{0, 0, [2]uint16{421, 422}, 0},
	{0, 0, [2]uint16{64, 30}, 0},
	{0, 0, [2]uint16{423, 424}, 0},
	{0, 0, [2]uint16{425, 426}, 0},
	{0, 0, [2]uint16{427, 428}, 0},
	{0, 0, [2]uint16{41, 429}, 0},
	{0, 0, [2]uint16{430, 431}, 0},
	{0, 0, [2]uint16{3, 432}, 0},
	{0, 0, [2]uint16{6, 433}, 0},
	{0, 0, [2]uint16{434, 435}, 0},
	{0, 0, [2]uint16{7, 436}, 0},
	{0, 0, [2]uint16{437, 438}, 0},
	{0, 0, [2]uint16{439, 11}, 0},
	{0, 0, [2]uint16{24, 440}, 0},
	{0, 0, [2]uint16{441, 442}, 0},
	{0, 0, [2]uint16{443, 444}, 0},
	{0, 0, [2]uint16{445, 446}, 0},
	{0, 0, [2]uint16{447, 9}, 0},
	{0, 0, [2]uint16{448, 83}, 0},
	{0, 0, [2]uint16{449, 18}, 0},
	{0, 0, [2]uint16{450, 451}, 0},
	{0, 0, [2]uint16{452, 453}, 0},
	{0, 0, [2]uint16{454, 12}, 0},
	{0, 0, [2]uint16{455, 456}, 0},
	{0, 0, [2]uint16{457, 16}, 0},
	{0, 0, [2]uint16{458, 459}, 0},
	{0, 0, [2]uint16{460, 10}, 0},
	{0, 0, [2]uint16{461, 29}, 0},
	{0, 0, [2]uint16{462, 40}, 0},
	{0, 0, [2]uint16{463, 5}, 0},
	{0, 0, [2]uint16{464, 465}, 0},
	{0, 0, [2]uint16{466, 467}, 0},
	{0, 0, [2]uint16{468, 469}, 0},
	{0, 0, [2]uint16{470, 471}, 0},
	{0, 0, [2]uint16{472, 473}, 0},
	{0, 0, [2]uint16{474, 475}, 0},
	{0, 0, [2]uint16{476, 477}, 0},
	{0, 0, [2]uint16{478, 479}, 0},
	{0, 0, [2]uint16{480, 481}, 0},
	{0, 0, [2]uint16{148, 482}, 0},
	{0, 0, [2]uint16{483, 484}, 0},
	{0, 0, [2]uint16{13, 485}, 0},
	{0, 0, [2]uint16{4, 486}, 0},
	{0, 0, [2]uint16{128, 487}, 0},
	{0, 0, [2]uint16{8, 488}, 0},
	{0, 0, [2]uint16{489, 490}, 0},
	{0, 0, [2]uint16{2, 491}, 0},
	{0, 0, [2]uint16{492, 493}, 0},
	{0, 0, [2]uint16{494, 495}, 0},
	{0, 0, [2]uint16{496, 497}, 0},
	{0, 0, [2]uint16{498, 1}, 0},
	{0, 0, [2]uint16{499, 500}, 0},
	{0, 0, [2]uint16{501, 502}, 0},
	{0, 0, [2]uint16{503, 504}, 0},
	{0, 0, [2]uint16{505, 506}, 0},
	{0, 0, [2]uint16{507, 508}, 0},
	{0, 0, [2]uint16{509, 510}, 0},
	{0, 0, [2]uint16{511, 0}, 0},
}
