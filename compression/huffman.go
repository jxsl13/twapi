package compression

import (
	"errors"
	"fmt"
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
	nodes             [HuffmanMaxNodes]node
	decodeLookupTable [HuffmanLookupTableSize]int
	startNodeIndex    int
	numNodes          int
}

type node struct {
	// symbol
	Bits    int
	NumBits int

	// don't use pointers for this. shorts are smaller so we can fit more data into the cache
	Left  int
	Right int

	// what the symbol represents
	Symbol int
}

type constructNode struct {
	nodeID    int
	frequency int
}

type byFrequencyDesc []*constructNode

func (a byFrequencyDesc) Len() int           { return len(a) }
func (a byFrequencyDesc) Swap(i, j int)      { *a[i], *a[j] = *a[j], *a[i] }
func (a byFrequencyDesc) Less(i, j int) bool { return a[i].frequency > a[j].frequency }

// NewHuffman expects a frequency table aka index -> symbol
// You can use the default one that can be found under protocol.FrequencyTable
func NewHuffman(frequencyTable [HuffmanMaxSymbols]int) (*Huffman, error) {
	m := make(map[int]int, len(frequencyTable))
	for idx, u := range frequencyTable {
		prevIdx, found := m[u]
		if found {
			return nil, fmt.Errorf("invalid frequency table: every element must be unique: found %d at %d and %d", u, prevIdx, idx)
		}
	}

	h := Huffman{}
	h.constructTree(frequencyTable)

	// build decode lookup table (LUT)
	for i := 0; i < HuffmanLookupTableSize; i++ {
		var (
			bits  = i
			index = h.startNodeIndex
		)

		var x int
		for x = 0; x < HuffmanLookupTableBits; x++ {
			if bits&1 != 0 {
				index = h.nodes[index].Right
			} else {
				index = h.nodes[index].Left
			}
			bits >>= 1

			child := h.nodes[index]
			if child.NumBits >= 0 {
				h.decodeLookupTable[i] = index
				break
			}
		}

		if x == HuffmanLookupTableBits {
			h.decodeLookupTable[i] = index
		}
	}
	return &h, nil
}

func (h *Huffman) Compress(data []byte) (compressed []byte) {
	compressed = make([]byte, 0, len(data))

	bits := 0
	bitCount := 0

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

/*
	def decompress(self, inp_buffer: bytearray, start_index: int = 0, size: int = -1):
        bits = 0
        bitcount = 0
        eof = self.nodes[HUFFMAN_EOF_SYMBOL]
        output = bytearray()

        src_index = start_index

        if size == -1:
            size = len(inp_buffer)
        else:
            size += src_index

        while True:
            node_i = None
            if bitcount >= HUFFMAN_LUTBITS:
                node_i = self.decode_lut[bits & HUFFMAN_LUTMASK]

            while bitcount < 24 and src_index != size:
                bits |= inp_buffer[src_index] << bitcount
                src_index += 1
                bitcount += 8

            if node_i is None:
                node_i = self.decode_lut[bits & HUFFMAN_LUTMASK]

            if self.nodes[node_i].numbits:
                bits >>= self.nodes[node_i].numbits
                bitcount -= self.nodes[node_i].numbits
            else:
                bits >>= HUFFMAN_LUTBITS
                bitcount -= HUFFMAN_LUTBITS

                while True:
                    if bits & 1:
                        node_i = self.nodes[node_i].right
                    else:
                        node_i = self.nodes[node_i].left

                    bitcount -= 1
                    bits >>= 1

                    if self.nodes[node_i].numbits:
                        break

                    if bitcount == 0:
                        raise ValueError("No more bits, decoding error")

            if self.nodes[node_i] == eof:
                break
            output.append(self.nodes[node_i].symbol)

        return output


*/

func (h *Huffman) Decompress(data []byte) (decompressed []byte, err error) {
	decompressed = make([]byte, 0, len(data))
	var (
		bits     = 0
		bitCount = 0
		eof      = h.nodes[HuffmanEOFSymbol]
		srcIndex = 0
		dataSize = len(data)
	)

	for {
		var nodeIndex int = -1
		if bitCount >= HuffmanLookupTableBits {
			nodeIndex = h.decodeLookupTable[bits&HuffmanLookupTableMask]
		}

		for bitCount < 24 && srcIndex < dataSize {
			bits |= int(data[srcIndex] << bitCount)
			srcIndex++
			bitCount += 8
		}

		if nodeIndex < 0 {
			nodeIndex = h.decodeLookupTable[bits&HuffmanLookupTableMask]
		}

		if n := h.nodes[nodeIndex]; n.NumBits > 0 {
			bits >>= n.NumBits
			bitCount -= n.NumBits
		} else {
			bits >>= HuffmanLookupTableBits
			bitCount -= HuffmanLookupTableBits

			for {
				if bits&1 != 0 {
					nodeIndex = h.nodes[nodeIndex].Right
				} else {
					nodeIndex = h.nodes[nodeIndex].Left
				}

				bitCount -= 1
				bits >>= 1

				if h.nodes[nodeIndex].NumBits > 0 {
					break
				}

				if bitCount == 0 {
					return decompressed, errors.New("decoding error: no more bits")
				}
			}
			if h.nodes[nodeIndex] == eof {
				break
			}
			decompressed = append(decompressed, byte(h.nodes[nodeIndex].Symbol))
		}
	}

	return decompressed, nil
}

func (h *Huffman) setBitsR(nodeIndex int, bits int, depth int) {
	var (
		n       = &h.nodes[nodeIndex]
		newBits int
	)

	if n.Right < 0xffff {
		newBits = bits | (1 << depth)
		h.setBitsR(n.Right, newBits, depth+1)
	}
	if n.Left < 0xffff {
		newBits = bits
		h.setBitsR(n.Left, newBits, depth+1)
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

	for i := 0; i < HuffmanMaxSymbols; i++ {
		n = &h.nodes[i]
		n.NumBits = 0xffffffff
		n.Symbol = i
		n.Left = 0xffff
		n.Right = 0xffff

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

		n := &h.nodes[h.numNodes]
		n1 := numNodesLeft - 1
		n2 := numNodesLeft - 2

		n.NumBits = 0
		n.Left = nodesLeft[n1].nodeID
		n.Right = nodesLeft[n2].nodeID

		freq1 := nodesLeft[n1].frequency
		freq2 := nodesLeft[n2].frequency

		nodesLeft[n2].nodeID = h.numNodes
		nodesLeft[n2].frequency = freq1 + freq2

		h.numNodes++
		numNodesLeft--
	}

	h.startNodeIndex = h.numNodes - 1
	h.setBitsR(h.startNodeIndex, 0, 0)
}
