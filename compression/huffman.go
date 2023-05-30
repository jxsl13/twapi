package compression

import (
	"errors"
	"sort"

	"github.com/jxsl13/twapi/protocol"
)

const (
	HuffmanEOFSymbol       = protocol.HuffmanEOFSymbol
	HuffmanMaxSymbols      = HuffmanEOFSymbol + 1
	HuffmanMaxNodes        = HuffmanMaxSymbols*2 - 1
	HuffmanLookupTableBits = 10
	HuffmanLookupTableSize = (1 << HuffmanLookupTableBits)
	HuffmanLookupTableMask = (HuffmanLookupTableSize - 1)
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
	NumBits uint32

	// don't use pointers for this. shorts are smaller so we can fit more data into the cache
	Leafs [2]uint16

	// what the symbol represents
	Symbol byte
}

type constructNode struct {
	nodeID    uint16
	frequency int
}

type byFrequencyDesc []*constructNode

func (a byFrequencyDesc) Len() int           { return len(a) }
func (a byFrequencyDesc) Swap(i, j int)      { *a[i], *a[j] = *a[j], *a[i] }
func (a byFrequencyDesc) Less(i, j int) bool { return a[i].frequency > a[j].frequency }

// NewHuffman expects a frequency table aka index -> symbol
// You can use the default one that can be found under protocol.FrequencyTable
func NewHuffman(frequencyTable [HuffmanMaxSymbols]int) (*Huffman, error) {

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
	return &h, nil
}

func (h *Huffman) Compress(data []byte) (compressed []byte) {
	compressed = make([]byte, 0, len(data))

	var (
		bits     uint32
		bitCount uint32
	)

	for _, x := range data {
		lookup := h.nodes[x]
		bits |= lookup.Bits << bitCount
		bitCount += lookup.NumBits

		for bitCount >= 8 {
			compressed = append(compressed, byte(bits))
			bits >>= 8
			bitCount -= 8
		}

	}
	lookupEOF := h.nodes[HuffmanEOFSymbol]
	bits |= lookupEOF.Bits << bitCount
	bitCount += lookupEOF.NumBits

	for bitCount >= 8 {
		compressed = append(compressed, byte(bits))
		bits >>= 8
		bitCount -= 8
	}

	if bitCount > 0 {
		compressed = append(compressed, byte(bits))
	}

	return compressed
}

func (h *Huffman) Decompress(data, decompressed []byte) (written int, err error) {

	var (
		src      = 0
		srcEnd   = len(data)
		dst      = 0
		dstEnd   = len(decompressed)
		bits     uint32
		bitCount uint32
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

func (h *Huffman) setBitsR(n *node, bits uint32, depth uint32) {
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

func (h *Huffman) sort(nodesLeftStorage []constructNode, nodesLeft []int) {
	if len(nodesLeftStorage) != len(nodesLeft) {
		panic("index list and object list length mismatch")
	}

	sort.Slice(nodesLeft, func(i, j int) bool {
		return nodesLeftStorage[i].frequency > nodesLeftStorage[j].frequency
	})
}

func (h *Huffman) constructTree(frequencyTable [HuffmanMaxSymbols]int) {

	var (
		nodesLeftStorage [HuffmanMaxSymbols]constructNode
		nodesLeft        [HuffmanMaxSymbols]*constructNode
		numNodesLeft     = HuffmanMaxSymbols

		n  *node
		ns *constructNode
	)

	for i := uint16(0); i < HuffmanMaxSymbols; i++ {
		n = &h.nodes[i]
		n.NumBits = 0xffffffff
		n.Symbol = byte(i) // TODO: EOF = 0
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

	h.numNodes = HuffmanMaxSymbols
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
