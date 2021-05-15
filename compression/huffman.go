package compression

import "errors"

type Node struct {
	// symbol
	Bits    uint
	NumBits uint

	// don't use pointers for this. shorts are smaller so we can fit more data into the cache
	Leafs [2]uint16

	// what the symbol represents
	Symbol byte
}

type huffmanConstructNode struct {
	NodeID    uint
	Frequency uint
}

type Huffman struct {
	Nodes     [HuffmanMaxNodes]Node
	DecodeLut [HuffmanLutsize]*Node
	StartNode *Node
	NumNodes  int
}

func (h *Huffman) setbitsR(node *Node, bits uint, depth uint) {
	if node.Leafs[1] != 0xffff {
		h.setbitsR(&h.Nodes[node.Leafs[1]], bits|(1<<depth), depth+1)
	}
	if node.Leafs[0] != 0xffff {
		h.setbitsR(&h.Nodes[node.Leafs[0]], bits, depth+1)
	}

	if node.NumBits != 0 {
		node.Bits = bits
		node.NumBits = depth
	}
}

func (h *Huffman) constructTree(frequencies []uint) {
	var nodesLeftStorage [HuffmanMaxSymbols]huffmanConstructNode
	var nodesLeft [HuffmanMaxSymbols]*huffmanConstructNode
	var numNodesLeft int = HuffmanMaxSymbols

	// add the symbols
	for i := uint(0); i < HuffmanMaxSymbols; i++ {
		h.Nodes[i].NumBits = 0xFFFFFFFF
		h.Nodes[i].Symbol = byte(i)
		h.Nodes[i].Leafs[0] = 0xffff
		h.Nodes[i].Leafs[1] = 0xffff

		if i == HuffmanEofSymbol {
			nodesLeftStorage[i].Frequency = 1
		} else {
			nodesLeftStorage[i].Frequency = frequencies[i]
		}
		nodesLeftStorage[i].NodeID = i
		nodesLeft[i] = &nodesLeftStorage[i]

	}

	h.NumNodes = HuffmanMaxSymbols

	// construct the table
	for numNodesLeft > 1 {
		// we can't rely on stdlib's qsort for this, it can generate different results on different implementations
		bubbleSort(nodesLeft[:])

		h.Nodes[h.NumNodes].NumBits = 0
		h.Nodes[h.NumNodes].Leafs[0] = uint16(nodesLeft[numNodesLeft-1].NodeID)
		h.Nodes[h.NumNodes].Leafs[1] = uint16(nodesLeft[numNodesLeft-2].NodeID)
		nodesLeft[numNodesLeft-2].NodeID = uint(h.NumNodes)
		nodesLeft[numNodesLeft-2].Frequency = nodesLeft[numNodesLeft-1].Frequency + nodesLeft[numNodesLeft-2].Frequency

		h.NumNodes++
		numNodesLeft--
	}

	// set start node
	h.StartNode = &h.Nodes[h.NumNodes-1]

	// build symbol bits
	h.setbitsR(h.StartNode, 0, 0)
}

func (h *Huffman) memZero() {
	memZeroNode(h.Nodes[:])
	memZeroNodePtr(h.DecodeLut[:])
	h.StartNode = nil
	h.NumNodes = 0
}

func (h *Huffman) Init(frequencies []uint) {
	// make sure to cleanout every thing
	h.memZero()

	// construct the tree
	if frequencies == nil {
		frequencies = freqTable
	}

	h.constructTree(frequencies)

	// build decode LUT
	for i := 0; i < HuffmanLutsize; i++ {
		bits := uint(i)
		k := 0
		node := h.StartNode
		for k = 0; k < HuffmanLutbits; k++ {
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

		if k == HuffmanLutbits {
			h.DecodeLut[i] = node
		}
	}

}

func (h *Huffman) Compress(input []byte, inputSize int, output *[]byte, outputSize int) int {
	if len(*output) < outputSize && cap(*output) >= outputSize {
		*output = (*output)[:outputSize]
	} else {
		(*output) = (*output)[:cap(*output)]
	}

	// setup buffer pointers
	pSrc := 0
	pSrcEnd := inputSize
	pDst := 0
	pDstEnd := outputSize

	// symbol variables
	Symbol := int(0)
	Bits := uint(0)
	Bitcount := uint(0)

	// this macro loads a symbol for a byte into bits and bitcount
	loadSymbol := func(symbol int) {
		Bits |= h.Nodes[symbol].Bits << Bitcount
		Bitcount += h.Nodes[symbol].NumBits
	}

	// this macro writes the symbol stored in bits and bitcount to the dst pointer
	write := func() error {
		for Bitcount >= 8 {
			(*output)[pDst] = byte(Bits & 0xff)
			pDst++

			if pDst == pDstEnd {
				return errors.New("failed to compress, reached end")
			}
			Bits >>= 8
			Bitcount -= 8
		}
		return nil
	}

	// make sure that we have data that we want to compress
	if inputSize != 0 {
		// {A} load the first symbol
		Symbol = int(input[pSrc])
		pSrc++

		for pSrc != pSrcEnd {
			// {B} load the symbol
			loadSymbol(Symbol)

			// {C} fetch next symbol, this is done here because it will reduce dependency in the code
			Symbol = int(input[pSrc])
			pSrc++

			// {B} write the symbol loaded at
			if err := write(); err != nil {
				return -1
			}
		}

		// write the last symbol loaded from {C} or {A} in the case of only 1 byte input buffer
		loadSymbol(Symbol)
		if err := write(); err != nil {
			return -1
		}
	}

	// write EOF symbol
	loadSymbol(HuffmanEofSymbol)
	if err := write(); err != nil {
		return -1
	}

	// write out the last bits
	(*output)[pDst] = byte(Bits)
	pDst++

	// resize slice to be the size of the result
	(*output) = (*output)[:pDst]

	// return the size of the output
	return pDst
}

func (h *Huffman) Decompress(input []byte, inputSize int, output *[]byte, outputSize int) int {
	if len(*output) < outputSize && cap(*output) >= outputSize {
		*output = (*output)[:outputSize]
	} else {
		(*output) = (*output)[:cap(*output)]
	}

	// setup buffer pointers
	pSrc := 0
	pSrcEnd := inputSize
	pDst := 0
	pDstEnd := outputSize

	Bits := 0
	Bitcount := 0

	pEof := &h.Nodes[HuffmanEofSymbol]
	pNode := (*Node)(nil)

	for {
		// {A} try to load a node now, this will reduce dependency at location {D}
		pNode = nil
		if Bitcount >= HuffmanLutbits {
			pNode = h.DecodeLut[Bits&HuffmanLutmask]
		}

		// {B} fill with new bits
		for Bitcount < 24 && pSrc != pSrcEnd {
			Bits |= int(input[pSrc]) << Bitcount
			pSrc++
			Bitcount += 8
		}

		// {C} load symbol now if we didn't that earlier at location {A}
		if pNode == nil {
			pNode = h.DecodeLut[Bits&HuffmanLutmask]
		}

		if pNode == nil {
			return -1
		}

		// {D} check if we hit a symbol already
		if pNode.NumBits != 0 {
			// remove the bits for that symbol
			Bits >>= pNode.NumBits
			Bitcount -= int(pNode.NumBits)
		} else {
			// remove the bits that the lut checked up for us
			Bits >>= HuffmanLutbits
			Bitcount -= HuffmanLutbits

			// walk the tree bit by bit
			for {
				// traverse tree
				pNode = &h.Nodes[pNode.Leafs[Bits&1]]

				// remove bit
				Bitcount--
				Bits >>= 1

				// check if we hit a symbol
				if pNode.NumBits != 0 {
					break
				}

				// no more bits, decoding error
				if Bitcount == 0 {
					return -1
				}
			}
		}

		// check for eof
		if pNode == pEof {
			break
		}

		// output character
		if pDst == pDstEnd {
			return -1
		}
		(*output)[pDst] = pNode.Symbol
		pDst++
	}

	// resize slice to be the size of the result
	(*output) = (*output)[:pDst]

	// return the size of the decompressed buffer
	return pDst
}
