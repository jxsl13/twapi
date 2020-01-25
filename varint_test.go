package main

import "testing"

func TestPack(t *testing.T) {
	var v VarInt
	v.Pack(0x3f)

	if v.Size() != 1 {
		t.Errorf("Expected size: 1 actual size: %d", v.Size())
	}

	v.Clear()
	v.Pack(0x40)

	if v.Size() != 2 {
		t.Errorf("Expected size: 2 actual size: %d", v.Size())
	}

	v.Clear()
	v.Pack(1048576 - 1) // 2^(6+7+7) -1

	if v.Size() != 3 {
		t.Errorf("Expected size: 3 actual size: %d", v.Size())
	}

	v.Clear()
	v.Pack(1048576) // 2^(6+7+7)

	if v.Size() != 4 {
		t.Errorf("Expected size: 4 actual size: %d", v.Size())
	}

	v.Clear()
	v.Pack(134217728 - 1) // 2^(6+7+7+7) -1

	if v.Size() != 4 {
		t.Errorf("Expected size: 4 actual size: %d", v.Size())
	}

	v.Clear()
	v.Pack(134217728) // 2^(6+7+7+7)

	if v.Size() != 5 {
		t.Errorf("Expected size: 5 actual size: %d", v.Size())
	}

	for i := -20000000; i < 20000000; i++ {
		v.Clear()
		v.Pack(i)
		j := v.Unpack()

		if i != j {
			t.Errorf("Packed %d, Unpacked to %d", i, j)
		}
	}

}
