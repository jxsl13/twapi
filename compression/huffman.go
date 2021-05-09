package compression

import (
	"errors"
)

var (
	frequenciesTable = initFrequenciesTable()
)

func initFrequenciesTable() [256 + 1]uint {
	return [257]uint{
		1 << 30, 4545, 2657, 431, 1950, 919, 444, 482, 2244, 617, 838, 542, 715, 1814, 304, 240, 754, 212, 647, 186,
		283, 131, 146, 166, 543, 164, 167, 136, 179, 859, 363, 113, 157, 154, 204, 108, 137, 180, 202, 176,
		872, 404, 168, 134, 151, 111, 113, 109, 120, 126, 129, 100, 41, 20, 16, 22, 18, 18, 17, 19,
		16, 37, 13, 21, 362, 166, 99, 78, 95, 88, 81, 70, 83, 284, 91, 187, 77, 68, 52, 68,
		59, 66, 61, 638, 71, 157, 50, 46, 69, 43, 11, 24, 13, 19, 10, 12, 12, 20, 14, 9,
		20, 20, 10, 10, 15, 15, 12, 12, 7, 19, 15, 14, 13, 18, 35, 19, 17, 14, 8, 5,
		15, 17, 9, 15, 14, 18, 8, 10, 2173, 134, 157, 68, 188, 60, 170, 60, 194, 62, 175, 71,
		148, 67, 167, 78, 211, 67, 156, 69, 1674, 90, 174, 53, 147, 89, 181, 51, 174, 63, 163, 80,
		167, 94, 128, 122, 223, 153, 218, 77, 200, 110, 190, 73, 174, 69, 145, 66, 277, 143, 141, 60,
		136, 53, 180, 57, 142, 57, 158, 61, 166, 112, 152, 92, 26, 22, 21, 28, 20, 26, 30, 21,
		32, 27, 20, 17, 23, 21, 30, 22, 22, 21, 27, 25, 17, 27, 23, 18, 39, 26, 15, 21,
		12, 18, 18, 27, 20, 18, 15, 19, 11, 17, 33, 12, 18, 15, 19, 18, 16, 26, 17, 18,
		9, 10, 25, 22, 22, 17, 20, 16, 6, 16, 15, 20, 14, 18, 24, 335, 1517,
	}
}

const (
	HuffmanEOFSymbol = 256

	HuffmanMaxSymbols = HuffmanEOFSymbol + 1
	HuffmanMaxNodes   = HuffmanMaxSymbols*2 - 1

	HuffmanLutBits = 10
	HuffmanLutSize = (1 << HuffmanLutBits)
	HuffmanLutMask = (HuffmanLutSize - 1)
)

type Node struct {
	// symbol
	Bits    uint
	NumBits uint

	// don't use pointers for this. shorts are smaller so we can fit more data into the cache
	Leafs [2]uint16

	// what the symbol represents
	Symbol byte
}

func NewHuffman() *Huffman {
	h := &Huffman{}
	h.Reset()
	h.Init(frequenciesTable[:])
	return h
}

type Huffman struct {
	Nodes     [HuffmanMaxNodes]Node
	DecodeLut [HuffmanLutSize]*Node
	StartNode *Node
	numNodes  int
}

func (h *Huffman) Reset() {
	memZeroNode(h.Nodes[:])
	memZeroNodePtr(h.DecodeLut[:])
	h.StartNode = nil
	h.numNodes = 0
}

func (h *Huffman) setBitsR(node *Node, bits uint, depth uint) {
	if node.Leafs[1] != 0xffff {
		h.setBitsR(&h.Nodes[node.Leafs[1]], bits|(1<<depth), depth+1)
	}
	if node.Leafs[0] != 0xffff {
		h.setBitsR(&h.Nodes[node.Leafs[0]], bits, depth+1)
	}

	if node.NumBits != 0 {
		node.Bits = bits
		node.NumBits = depth
	}
}

type huffmanConstructNode struct {
	NodeID    uint16
	Frequency uint
}

func (h *Huffman) constructTree(frequencies []uint) {
	nodesLeftStorage := [HuffmanMaxSymbols]huffmanConstructNode{}
	nodesLeft := [HuffmanMaxSymbols]*huffmanConstructNode{}

	numNodesLeft := HuffmanMaxSymbols

	// add the symbols
	for i := 0; i < HuffmanMaxSymbols; i++ {
		h.Nodes[i].NumBits = 0xFFFFFFFF
		h.Nodes[i].Symbol = byte(i)
		h.Nodes[i].Leafs[0] = 0xffff
		h.Nodes[i].Leafs[1] = 0xffff

		if i == HuffmanEOFSymbol {
			nodesLeftStorage[i].Frequency = 1
		} else {
			nodesLeftStorage[i].Frequency = frequencies[i]
		}
		nodesLeftStorage[i].NodeID = uint16(i)
		nodesLeft[i] = &nodesLeftStorage[i]

	}

	numNodes := uint16(HuffmanMaxSymbols)

	// construct the table
	for numNodesLeft > 1 {
		// we can't rely on stdlib's qsort for this, it can generate different results on different implementations

		bubbleSort(nodesLeft[:numNodesLeft])

		h.Nodes[numNodes].NumBits = 0
		h.Nodes[numNodes].Leafs[0] = nodesLeft[numNodesLeft-1].NodeID
		h.Nodes[numNodes].Leafs[1] = nodesLeft[numNodesLeft-2].NodeID
		nodesLeft[numNodesLeft-2].NodeID = numNodes
		nodesLeft[numNodesLeft-2].Frequency = nodesLeft[numNodesLeft-1].Frequency + nodesLeft[numNodesLeft-2].Frequency

		numNodes++
		numNodesLeft--
	}

	// set start node
	h.StartNode = &h.Nodes[numNodes-1]

	// build symbol bits
	h.setBitsR(h.StartNode, 0, 0)
}

//Function: huffman_init
//		Inits the compressor/decompressor.
// Parameters:
// 		huff - Pointer to the state to init
// 		frequencies - A pointer to an array of 256 entries of the frequencies of the bytes
//
// Remarks:
// 		- Does no allocation what so ever.
// 		- You don't have to call any cleanup functions when you are done with it
func (h *Huffman) Init(frequencies []uint) {
	// make sure to cleanout every thing
	h.Reset()

	// construct the tree
	if frequencies == nil {
		frequencies = frequenciesTable[:]
	}

	h.constructTree(frequencies)

	// build decode LUT
	for i := 0; i < HuffmanLutSize; i++ {
		bits := uint(i)
		k := 0
		node := h.StartNode
		for k = 0; k < HuffmanLutBits; k++ {

			node = &h.Nodes[node.Leafs[bits&1]]
			bits >>= 1

			if node == nil {
				break
			}

			if node.NumBits != 0 {
				h.DecodeLut[i] = node
				break
			}
		}

		if k == HuffmanLutBits {
			h.DecodeLut[i] = node
		}
	}

}

// 	Compress compresses a buffer and outputs a compressed buffer.
// Parameters:
// 		input - Buffer to compress
// 		input_size - Size of the buffer to compress
// 		output - Buffer to put the compressed data into
// 		output_size - Size of the output buffer
// Returns:
// 	Returns the size of the compressed data. Negative value on failure.
func (h *Huffman) Compress(input []byte) (output []byte, err error) {

	// symbol variables
	Bits := uint(0)
	Bitcount := uint(0)

	// setup buffer indices
	src := 0
	srcEnd := len(input)

	// convenience function like the macro in the C code
	loadSymbol := func(Sym uint) {
		Bits |= h.Nodes[Sym].Bits << Bitcount
		Bitcount += h.Nodes[Sym].NumBits
	}

	// convenience function like the makro in the C code
	write := func() error {
		for Bitcount >= 8 {
			value := byte(Bits & 0xff)
			output = append(output, value)

			Bits >>= 8
			Bitcount -= 8
		}
		return nil
	}

	// make sure that we have data that we want to compress
	if len(input) > 0 {
		// {A} load the first symbol
		Symbol := uint(input[src])
		src++

		for src != srcEnd {
			// {B} load the symbol
			loadSymbol(Symbol)

			// {C} fetch next symbol, this is done here because it will reduce dependency in the code
			Symbol = uint(input[src])
			src++

			// {B} write the symbol loaded at
			if err := write(); err != nil {
				return nil, err
			}
		}

		// write the last symbol loaded from {C} or {A} in the case of only 1 byte input buffer
		loadSymbol(Symbol)
		if err := write(); err != nil {
			return nil, err
		}
	}

	// write EOF symbol
	loadSymbol(HuffmanEOFSymbol)
	if err := write(); err != nil {
		return nil, err
	}

	// write out the last bits
	output = append(output, byte(Bits))

	// return the size of the output
	return output, nil

}

// Decompress decompresses a buffer
// Parameters:
// 	input - Buffer to decompress
// 	input_size - Size of the buffer to decompress
// 	output - Buffer to put the uncompressed data into
// 	output_size - Size of the output buffer
//
// Returns:
// Returns the size of the uncompressed data. Negative value on failure.
func (h *Huffman) Decompress(input []byte) (output []byte, err error) {

	output = make([]byte, 0, len(input)*2)

	// setup buffer pointers
	src := 0
	srcEnd := len(input)

	Bits := uint(0)
	Bitcount := uint(0)

	EOF := &h.Nodes[HuffmanEOFSymbol]
	node := (*Node)(nil)

	for {
		// {A} try to load a node now, this will reduce dependency at location {D}
		node = nil
		if Bitcount >= HuffmanLutBits {
			node = h.DecodeLut[Bits&HuffmanLutMask]
		}

		// {B} fill with new bits
		for Bitcount < 24 && src != srcEnd {
			Bits |= uint(input[src] << Bitcount)
			src++
			Bitcount += 8
		}

		// {C} load symbol now if we didn't that earlier at location {A}
		if node == nil {
			node = h.DecodeLut[Bits&HuffmanLutMask]
		}
		// if node still nil
		if node == nil {
			return nil, errors.New("decompression failed: node is nil")
		}

		// {D} check if we hit a symbol already
		if node.NumBits != 0 {

			// remove the bits for that symbol
			Bits >>= node.NumBits
			Bitcount -= node.NumBits

		} else {

			// remove the bits that the lut checked up for us
			Bits >>= HuffmanLutBits
			Bitcount -= HuffmanLutBits

			// walk the tree bit by bit
			for {
				// traverse tree
				node = &h.Nodes[node.Leafs[Bits&1]]

				// remove bit
				Bitcount--
				Bits >>= 1

				// check if we hit a symbol
				if node.NumBits != 0 {
					break
				}

				// no more bits, decoding error
				if Bitcount == 0 {
					return nil, errors.New("decompression failed: no more bits")
				}
			}
		}

		// check for eof
		if node == EOF {
			break
		}

		output = append(output, node.Symbol)
	}

	// return the size of the decompressed buffer
	return output, nil
}
