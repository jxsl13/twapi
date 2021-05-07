package compression

func memZeroNode(a []Node) {
	if len(a) == 0 {
		return
	}
	a[0] = Node{}
	for bp := 1; bp < len(a); bp *= 2 {
		copy(a[bp:], a[:bp])
	}
}

func memZeroNodePtr(a []*Node) {
	if len(a) == 0 {
		return
	}
	a[0] = nil
	for bp := 1; bp < len(a); bp *= 2 {
		copy(a[bp:], a[:bp])
	}
}

func bubbleSort(list []*huffmanConstructNode) {
	changed := true
	var temp *huffmanConstructNode
	size := len(list)
	for changed {
		changed = false
		for i := 0; i < size-1; i++ {

			if list[i].Frequency < list[i+1].Frequency {
				temp = list[i]
				list[i] = list[i+1]
				list[i+1] = temp
				changed = true
			}
		}
		size--
	}
}
