package main

import (
	"math"
	"math/bits"
	"math/rand"
	"testing"
	"time"
	"unsafe"
)

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

func TestMultiplePacks(t *testing.T) {
	var v VarInt
	v.Clear()

	randoNumbers := 10000000
	v.Grow(randoNumbers)

	seedSource := rand.NewSource(time.Now().UnixNano())
	randomNumberGenerator := rand.New(seedSource)

	numbers := make([]int, randoNumbers)
	sign := 0

	maxAllowedValue := int64(math.Pow(2, 55)) // 2^55 is the max range of this compression format

	// generate random numbers
	for idx := range numbers {

		if idx%2 == 0 {
			sign = -1
		} else {
			sign = 1
		}

		value := sign * int(randomNumberGenerator.Int63n(maxAllowedValue)) // max values should be below 2^55
		numbers[idx] = value
		v.Pack(value)
	}

	// compare to unpacked values
	errors := 0
	unpackedValue := 0
	for idx, expectedValue := range numbers {
		unpackedValue = v.Unpack()

		if expectedValue != unpackedValue {
			errors++
			if errors > 100 {
				break
			}
			if expectedValue >= math.MaxInt32 || expectedValue <= math.MinInt32 {
				length := bits.Len(uint(expectedValue))
				t.Errorf("%d %d Expected(%d): %d Unpacked: %d", 64, idx, length, expectedValue, unpackedValue)
			} else {
				t.Errorf("%d %d Expected: %d Unpacked: %d", 32, idx, expectedValue, unpackedValue)
			}

		}
	}
}

func TestPackLong(t *testing.T) {
	var v VarInt
	v.Clear()

	toPack := int(3.6028797e16) // 2^55
	v.Pack(toPack)

	value := v.Unpack()
	t.Logf("byte needed: %d,packed: %d unpacked: %d", unsafe.Sizeof(toPack), toPack, value)
	t.Logf("")

	if value != toPack {
		t.Error("value mismatch")
	}

}

// func TestPackLibrary(t *testing.T) {
// 	// same as the Varint from the golang library
// 	toPack := int64(7.6028797e18) // 2^55

// 	buf := make([]byte, 10)
// 	binary.PutVarint(buf, toPack)

// 	v := VarInt{buf}

// 	value := v.Unpack()

// 	t.Logf("byte needed: %d,packed: %d unpacked: %d", unsafe.Sizeof(toPack), toPack, value)
// 	t.Logf("")

// 	if value != int(toPack) {
// 		t.Error("value mismatch")
// 	}
// }

// func TestUnpackLibrary(t *testing.T) {
// 	toPack := int(7.6028797e18) // 2^55
// 	expected := int64(toPack)

// 	v := VarInt{}
// 	v.Pack(toPack)

// 	buffer := bytes.NewBuffer(v.Compressed)
// 	result, err := binary.ReadVarint(buffer)

// 	if err != nil {
// 		t.Error(err)
// 		return
// 	}

// 	if expected != result {
// 		t.Errorf("expected: %d received: %d", expected, result)
// 	}
// }
