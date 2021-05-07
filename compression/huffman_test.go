package compression

import (
	"fmt"
	"testing"
)

func Test_bubbleSort(t *testing.T) {
	type testStruct struct {
		name string
		list []*huffmanConstructNode
	}

	tests := []testStruct{}

	// 10! = 3628800 unique lists
	initialList := []*huffmanConstructNode{
		{0, 1},
		{0, 2},
		{0, 3},
		{0, 4},
		{0, 5},
		{0, 6},
		{0, 7},
		{0, 8},
		{0, 9},
		{0, 10},
	}

	// create all possible permutations for the list
	allPermutations := permutate(initialList)

	// put all permutations into tests
	for idx, permutation := range allPermutations {
		tests = append(tests,
			testStruct{
				fmt.Sprintf("#%d", idx+1),
				permutation,
			})
	}

	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// sort permutation
			bubbleSort(tt.list)

			// resulting permutation must have ordered frequencies
			for idx, value := range tt.list {
				if idx > 0 && tt.list[idx-1].Frequency < value.Frequency {
					t.Errorf("idx: %d = %d, idx: %d = %d",
						idx-1,
						tt.list[idx-1].Frequency,
						idx,
						value.Frequency,
					)

				}
			}
		})
	}
}

func permutate(a []*huffmanConstructNode) [][]*huffmanConstructNode {
	var res [][]*huffmanConstructNode
	calPermutation(a, &res, 0)
	return res
}
func calPermutation(arr []*huffmanConstructNode, res *[][]*huffmanConstructNode, k int) {
	for i := k; i < len(arr); i++ {
		swap(arr, i, k)
		calPermutation(arr, res, k+1)
		swap(arr, k, i)
	}
	if k == len(arr)-1 {
		r := make([]*huffmanConstructNode, len(arr))
		copy(r, arr)
		*res = append(*res, r)
		return
	}
}
func swap(arr []*huffmanConstructNode, i, k int) {
	arr[i], arr[k] = arr[k], arr[i]
}

func TestHuffman_Compress_Decompress(t *testing.T) {
	huffman := Huffman{}
	huffman.Init(nil)

	outputBuffer := &[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	type args struct {
		input  []byte
		output *[]byte
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{"#1", args{[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}, outputBuffer}, 10, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got, err := huffman.Compress(tt.args.input, tt.args.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("Huffman.Compress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Huffman.Compress() = %v, want %v", got, tt.want)
			}
		})
	}
}
