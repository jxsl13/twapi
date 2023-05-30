package compression

import (
	"errors"
	"fmt"
	"sort"

	"github.com/jxsl13/twapi/protocol"
)

const (
	HuffmanEOFSymbol  = protocol.HuffmanEOFSymbol
	HuffmanMaxSymbols = protocol.HuffmanMaxSymbols

	HuffmanMaxNodes        = (HuffmanMaxSymbols)*2 + 1 // +1 for additional EOF symbol
	HuffmanLookupTableBits = 10
	HuffmanLookupTableSize = (1 << HuffmanLookupTableBits)
	HuffmanLookupTableMask = (HuffmanLookupTableSize - 1)
)

var (
	ErrHuffmanCompress   = errors.New("compression error")
	ErrHuffmanDecompress = errors.New("decompression error")
)

type Huffman struct {
	nodes     [HuffmanMaxNodes]node
	decodeLut [HuffmanLookupTableSize]*node
	startNode *node
	numNodes  uint16
}

type node struct {
	// symbol
	Bits    uint32
	NumBits uint8

	// don't use pointers for this. shorts are smaller so we can fit more data into the cache
	Leafs [2]uint16

	// what the symbol represents
	Symbol byte
}

type constructNode struct {
	nodeID    uint16
	frequency uint32
}

type byFrequencyDesc []*constructNode

func (a byFrequencyDesc) Len() int           { return len(a) }
func (a byFrequencyDesc) Swap(i, j int)      { *a[i], *a[j] = *a[j], *a[i] }
func (a byFrequencyDesc) Less(i, j int) bool { return a[i].frequency > a[j].frequency }

// NewHuffman expects a frequency table aka index -> symbol
// You can use the default one that can be found under protocol.FrequencyTable
// You can put the frequency at index  HuffmanMaxSymbols -1 to the value 1.
// It is the EOF value which will ne overwritten anyway.
func NewHuffman(frequencyTable [HuffmanMaxSymbols]uint32) *Huffman {

	h := Huffman{}
	h.constructTree(frequencyTable)

	// build decode lookup table (LUT)
	for i := 0; i < HuffmanLookupTableSize; i++ {
		var (
			bits uint32 = uint32(i)
			k    int
			n    = h.startNode
		)

		for k = 0; k < HuffmanLookupTableBits; k++ {
			n = &h.nodes[n.Leafs[bits&1]]
			bits >>= 1

			if n.NumBits > 0 {
				h.decodeLut[i] = n
				break
			}
		}

		if k == HuffmanLookupTableBits {
			h.decodeLut[i] = n
		}

	}
	return &h
}

// Compress compresses in data to the compressed slice.
// 'compressed' must be preallocated with enough space to fit the result.
func (h *Huffman) Compress(data, compressed []byte) (written int, err error) {

	var (
		dst      = 0
		dstEnd   = len(compressed)
		bits     uint32
		bitCount uint8
	)

	for _, x := range data {
		lookup := h.nodes[x]
		bits |= lookup.Bits << bitCount
		bitCount += lookup.NumBits

		for bitCount >= 8 {
			if dst == dstEnd {
				return dst, fmt.Errorf("%w: : compression buffer too small", ErrHuffmanCompress)
			}
			compressed[dst] = byte(bits)
			dst++
			bits >>= 8
			bitCount -= 8
		}

	}
	lookupEOF := h.nodes[HuffmanEOFSymbol]
	bits |= lookupEOF.Bits << bitCount
	bitCount += lookupEOF.NumBits

	for bitCount >= 8 {
		if dst == dstEnd {
			return dst, fmt.Errorf("%w: : compression buffer too small", ErrHuffmanCompress)
		}
		compressed[dst] = byte(bits)
		dst++
		bits >>= 8
		bitCount -= 8
	}

	if bitCount > 0 {
		if dst == dstEnd {
			return dst, fmt.Errorf("%w: : compression buffer too small", ErrHuffmanCompress)
		}
		compressed[dst] = byte(bits)
		dst++
	}

	return dst, nil
}

// Decompress decompresses 'data' and writes the result into 'decompressed'.
// The decompressed slice must be preallocated to fit the decompressed data.
func (h *Huffman) Decompress(data, decompressed []byte) (written int, err error) {

	var (
		src      = 0
		srcEnd   = len(data)
		dst      = 0
		dstEnd   = len(decompressed)
		bits     uint32
		bitCount uint8
		eof      *node = &h.nodes[HuffmanEOFSymbol]
		n        *node = nil
	)

	for {
		n = nil
		if bitCount >= HuffmanLookupTableBits {
			n = h.decodeLut[bits&HuffmanLookupTableMask]
		}

		for bitCount < 24 && src < srcEnd {
			bits |= uint32(data[src]) << bitCount
			src++
			bitCount += 8
		}

		if n == nil {
			n = h.decodeLut[bits&HuffmanLookupTableMask]
		}

		if n == nil {
			return dst, errors.New("decoding error: symbol not found in lookup table")
		}

		if n.NumBits > 0 {
			bits >>= n.NumBits
			bitCount -= n.NumBits
		} else {
			bits >>= HuffmanLookupTableBits
			bitCount -= HuffmanLookupTableBits

			// walk the tree bit by bit
			for {
				// traverse tree
				n = &h.nodes[n.Leafs[bits&1]]

				// remove bit
				bitCount--
				bits >>= 1

				// check if we hit a symbol
				if n.NumBits > 0 {
					break
				}

				if bitCount == 0 {
					return dst, errors.New("decoding error: symbol not found in tree")
				}
			}
		}

		if n == eof {
			break
		}

		if dst == dstEnd {
			return dst, errors.New("decompression failed: not enough space in decompression buffer")
		}

		decompressed[dst] = n.Symbol
		dst++
	}

	return dst, nil
}

func (h *Huffman) setBitsR(n *node, bits uint32, depth uint8) {
	var (
		newBits uint32
		left    = n.Leafs[0]
		right   = n.Leafs[1]
	)

	if right < 0xffff {
		newBits = bits | (1 << depth)
		h.setBitsR(&h.nodes[right], newBits, depth+1)
	}
	if left < 0xffff {
		newBits = bits
		h.setBitsR(&h.nodes[left], newBits, depth+1)
	}

	if n.NumBits > 0 {
		n.Bits = bits
		n.NumBits = depth
	}
}

func (h *Huffman) constructTree(frequencyTable [HuffmanMaxSymbols]uint32) {

	var (
		// +1 for additional EOF symbol
		nodesLeftStorage [HuffmanMaxSymbols + 1]constructNode
		nodesLeft        [HuffmanMaxSymbols + 1]*constructNode
		numNodesLeft     = HuffmanMaxSymbols + 1

		n  *node
		ns *constructNode
	)

	// +1 for EOF symbol
	for i := uint16(0); i < HuffmanMaxSymbols+1; i++ {
		n = &h.nodes[i]
		n.NumBits = 0xff
		n.Symbol = byte(i)
		n.Leafs[0] = 0xffff
		n.Leafs[1] = 0xffff

		ns = &nodesLeftStorage[i]
		if i == HuffmanEOFSymbol {
			ns.frequency = 1
		} else {
			ns.frequency = frequencyTable[i]
		}
		ns.nodeID = i
		nodesLeft[i] = ns
	}

	h.numNodes = HuffmanMaxSymbols + 1 // +1 for EOF symbol
	for numNodesLeft > 1 {

		sort.Stable(byFrequencyDesc(nodesLeft[:numNodesLeft]))

		n = &h.nodes[h.numNodes]
		n1 := numNodesLeft - 1
		n2 := numNodesLeft - 2

		n.NumBits = 0
		n.Leafs[0] = nodesLeft[n1].nodeID
		n.Leafs[1] = nodesLeft[n2].nodeID

		freq1 := nodesLeft[n1].frequency
		freq2 := nodesLeft[n2].frequency

		nodesLeft[n2].nodeID = h.numNodes
		nodesLeft[n2].frequency = freq1 + freq2

		h.numNodes++
		numNodesLeft--
	}

	h.startNode = n
	h.setBitsR(n, 0, 0)
}
