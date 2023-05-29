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
	decodeLookupTable [HuffmanLookupTableSize]*node
	startNode         *node
	numNodes          int
}

type node struct {
	// symbol
	Bits    uint32
	NumBits uint32

	// don't use pointers for this. shorts are smaller so we can fit more data into the cache
	Leafs [2]uint16

	// what the symbol represents
	Symbol uint16
}

type constructNode struct {
	nodeID    uint16
	frequency uint32
}

// NewHuffman expects a frequency table aka index -> symbol
// You can use the default one that can be found under protocol.FrequencyTable
func NewHuffman(frequencyTable [HuffmanMaxSymbols]uint32) (*Huffman, error) {
	m := make(map[uint32]int, len(frequencyTable))
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
			bits uint = 0
			k    int
		)
		node := h.startNode
		for k = 0; k < HuffmanLookupTableBits; k++ {
			node = &h.nodes[node.Leafs[bits&1]]
			bits >>= 1
			if node == nil {
				// TODO: this cannot happen
				break
			}

			if node.NumBits != 0 {
				h.decodeLookupTable[i] = node
				break
			}
		}

		if k == HuffmanLookupTableBits {
			h.decodeLookupTable[i] = node
		}
	}
	return &h, nil
}

func (h *Huffman) Compress(data []byte, buf []byte) (written int, err error) {
	if len(data) > 0 && len(buf) == 0 {
		return 0, fmt.Errorf("data to compress %d buf no space in buffer", len(data))
	}

	var (
		bits     uint32
		bitCount uint32
		dataEnd  = len(data) - 1
		bufEnd   = len(buf) - 1
		i        = 0
	)

	if len(data) > 0 {
		symbol := data[i]
		i++

		for i < dataEnd {
			// load symbol
			bits |= h.nodes[symbol].Bits << bitCount
			bitCount += h.nodes[symbol].NumBits
			symbol = data[i]
			i++

			// write symbol
			if i > bufEnd {
				return i, fmt.Errorf("trying to write more data (%d) than the buffer(%d) can hold", len(data), len(buf))
			}
			buf[i] = byte(bits & 0xFF)
			i++
			bits >>= 8
			bitCount -= 8
		}
	}

	// load EOF
	bits |= h.nodes[HuffmanEOFSymbol].Bits << bitCount
	bitCount += h.nodes[HuffmanEOFSymbol].NumBits
	// write EOF
	if i > bufEnd {
		return i, fmt.Errorf("trying to write more data (%d) than the buffer(%d) can hold", len(data), len(buf))
	}
	buf[i] = byte(bits & 0xff)
	i++
	bits >>= 8
	bitCount -= 8

	buf[i] = byte(bits)
	i++
	return i, nil
}

func (h *Huffman) Decompress(data []byte, buf []byte) (read int, err error) {
	var (
		bits     uint32
		bitCount uint32
		eofNode  = &h.nodes[HuffmanEOFSymbol]
		node     *node
		src      = 0
		srcEnd   = len(data) - 1
		dst      = 0
		dstEnd   = len(buf) - 1
	)

	for {
		node = nil
		if bitCount >= HuffmanLookupTableBits {
			node = h.decodeLookupTable[bits&HuffmanLookupTableMask]
		}

		for bitCount < 24 && src <= srcEnd {
			bits |= uint32(data[src]) << bitCount
			bitCount += 8
			src++
		}

		if node == nil {
			node = h.decodeLookupTable[bits&HuffmanLookupTableMask]
		}

		if node == nil {
			return dst, errors.New("decompression error: node is nil")
		}

		if node.NumBits > 0 {
			bits >>= node.NumBits
			bitCount -= node.NumBits
		} else {
			bits >>= HuffmanLookupTableBits
			bitCount -= HuffmanLookupTableBits

			for {
				node = &h.nodes[node.Leafs[bits&1]]
				bitCount--
				bits >>= 1

				if node.NumBits > 0 {
					break
				}
				if bitCount == 0 {
					return dst, errors.New("decompression error: invalid compression")
				}
			}
		}

		if node == eofNode {
			break
		}

		if dst == dstEnd {
			return dst, errors.New("decompression error: output buffer too small: returned ")
		}

		buf[dst] = byte(node.Symbol)
		dst++
	}

	return dst, nil
}

/*

	void CHuffman::Setbits_r(CNode *pNode, int Bits, unsigned Depth)
	{
		if(pNode->m_aLeafs[1] != 0xffff)
			Setbits_r(&m_aNodes[pNode->m_aLeafs[1]], Bits|(1<<Depth), Depth+1);
		if(pNode->m_aLeafs[0] != 0xffff)
			Setbits_r(&m_aNodes[pNode->m_aLeafs[0]], Bits, Depth+1);

		if(pNode->m_NumBits)
		{
			pNode->m_Bits = Bits;
			pNode->m_NumBits = Depth;
		}
	}

*/

func (h *Huffman) setBitsR(n *node, bits uint32, depth uint32) {

	var (
		leaf    *node
		newBits uint32
		left    = n.Leafs[0]
		right   = n.Leafs[1]
	)

	if right != 0xffff {
		leaf = &h.nodes[right]
		newBits = bits | (1 << depth)
		h.setBitsR(leaf, newBits, depth+1)
	}
	if left != 0xffff {
		leaf = &h.nodes[left]
		newBits = bits
		h.setBitsR(leaf, newBits, depth+1)
	}

	if n.NumBits > 0 {
		n.Bits = bits
		n.NumBits = depth
	}
}

type byFrequency []*constructNode

func (a byFrequency) Len() int           { return len(a) }
func (a byFrequency) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byFrequency) Less(i, j int) bool { return a[i].frequency < a[j].frequency }

func (h Huffman) sort(nodes []*constructNode) {
	// from small to big frequency
	sort.Sort(byFrequency(nodes))
}

func (h *Huffman) constructTree(frequencyTable [HuffmanMaxSymbols]uint32) {

	var (
		nodesLeftStorage [HuffmanMaxSymbols]constructNode
		nodesLeft        [HuffmanMaxSymbols]*constructNode
		numNodesLeft     = HuffmanMaxSymbols
	)

	/*
		// add the symbols
		for(int i = 0; i < HUFFMAN_MAX_SYMBOLS; i++)
		{
			m_aNodes[i].m_NumBits = 0xFFFFFFFF;
			m_aNodes[i].m_Symbol = i;
			m_aNodes[i].m_aLeafs[0] = 0xffff;
			m_aNodes[i].m_aLeafs[1] = 0xffff;

			if(i == HUFFMAN_EOF_SYMBOL)
				aNodesLeftStorage[i].m_Frequency = 1;
			else
				aNodesLeftStorage[i].m_Frequency = pFrequencies[i];
			aNodesLeftStorage[i].m_NodeId = i;
			apNodesLeft[i] = &aNodesLeftStorage[i];

		}
	*/

	// 0 through 256
	for i := uint16(0); i < HuffmanMaxSymbols; i++ {
		h.nodes[i].NumBits = 0xffffffff
		h.nodes[i].Symbol = i // TODO: assignment invalid value 256 to byte
		h.nodes[i].Leafs[0] = 0xffff
		h.nodes[i].Leafs[1] = 0xffff

		if i == HuffmanEOFSymbol {
			nodesLeftStorage[i].frequency = 1
		} else {
			nodesLeftStorage[i].frequency = frequencyTable[i]
		}
		nodesLeftStorage[i].nodeID = i
		nodesLeft[i] = &nodesLeftStorage[i]
	}

	// m_NumNodes = HUFFMAN_MAX_SYMBOLS;
	h.numNodes = HuffmanMaxSymbols

	/*
		// construct the table
		while(NumNodesLeft > 1)
		{
			// we can't rely on stdlib's qsort for this, it can generate different results on different implementations
			BubbleSort(apNodesLeft, NumNodesLeft);

			m_aNodes[m_NumNodes].m_NumBits = 0;
			m_aNodes[m_NumNodes].m_aLeafs[0] = apNodesLeft[NumNodesLeft-1]->m_NodeId;
			m_aNodes[m_NumNodes].m_aLeafs[1] = apNodesLeft[NumNodesLeft-2]->m_NodeId;
			apNodesLeft[NumNodesLeft-2]->m_NodeId = m_NumNodes;
			apNodesLeft[NumNodesLeft-2]->m_Frequency = apNodesLeft[NumNodesLeft-1]->m_Frequency + apNodesLeft[NumNodesLeft-2]->m_Frequency;

			m_NumNodes++;
			NumNodesLeft--;
		}

	*/
	for numNodesLeft > 1 {

		h.sort(nodesLeft[:numNodesLeft])

		var (
			n  *node          = &h.nodes[h.numNodes]
			n1 *constructNode = nodesLeft[numNodesLeft-1]
			n2 *constructNode = nodesLeft[numNodesLeft-2]
		)

		n.NumBits = 0
		n.Leafs[0] = n1.nodeID
		n.Leafs[1] = n2.nodeID

		n2.nodeID = uint16(h.numNodes)
		n2.frequency = n1.frequency + n2.frequency

		h.numNodes++
		numNodesLeft--
	}

	/*

		// set start node
		m_pStartNode = &m_aNodes[m_NumNodes-1];

		// build symbol bits
		Setbits_r(m_pStartNode, 0, 0);
	*/

	h.startNode = &h.nodes[h.numNodes-1]
	h.setBitsR(h.startNode, 0, 0)
}
