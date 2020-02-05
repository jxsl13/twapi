package twapi

import "unsafe"

// VarInt is used to compress integers in a variable length format.
// Format: ESDDDDDD EDDDDDDD EDD... Extended, Data, Sign
// E: is next byte part of the current integer
// S: Sign of integer
// Data, Integer bits that follow the sign
type VarInt struct {
	Compressed []byte
}

// NewVarIntFrom Bytes allows for a creation if a buffer based on a preexisting buffer
func NewVarIntFrom(bytes []byte) VarInt {
	return VarInt{bytes}
}

// Size returns the length of the data.
// not its capacity
func (v *VarInt) Size() int {
	if v.Compressed == nil {
		v.Clear()
	}
	return len(v.Compressed)
}

// Data returns the Compressed data slice.
func (v *VarInt) Data() []byte {
	if v.Compressed == nil {
		v.Clear()
	}
	return v.Compressed
}

// Clear clears the internal Compressed buffer
func (v *VarInt) Clear() {
	v.Compressed = make([]byte, 0, 5)
}

// Grow increases size of the underlying array to fit another n elements
func (v *VarInt) Grow(n int) {
	if v.Compressed == nil {
		v.Compressed = make([]byte, 0, n)
	} else {

	}
	newBuffer := make([]byte, len(v.Compressed), cap(v.Compressed)+n)
	copy(newBuffer, v.Compressed)

	v.Compressed = newBuffer
}

// Unpack the wrapped Compressed buffer
func (v *VarInt) Unpack() (value int) {

	if v.Compressed == nil {
		v.Clear()
	}

	if len(v.Compressed) == 0 {
		panic("Error: VarInt is empty, please check the size before trying to Unpack")
	}

	intSize := int(unsafe.Sizeof(value))

	index := 0
	data := v.Compressed[:]

	// handle first byte (most right side)
	sign := int((data[index] >> 6) & 0b00000001)
	value = int(data[index] & 0b00111111)

	// handle 2nd - nth byte
	for i := 0; i < intSize-1; i++ {
		if data[index] < 0b10000000 {
			break
		}
		index++
		value |= int(data[index]&0b01111111) << (6 + 7*i)
	}

	index++
	value ^= -sign // if(sign) value = ~(value)

	// continue walking over the buffer
	v.Compressed = v.Compressed[index:]
	return
}

// Pack a value to internal buffer
func (v *VarInt) Pack(value int) {
	if v.Compressed == nil {
		v.Clear()
	}

	if value < -3.6028797e16 || 3.6028797e16 < value {
		panic("ERROR: value to Pack is out of bounds, should lie within range [-2^55:2^55]]")
	}

	intSize := unsafe.Sizeof(value)
	maxBufferSize := uintptr(64/(intSize*8)) + 2 + (intSize - 1) // (sign bit + extend bit) + (n-1) * (extend bit)

	// buffer
	data := make([]byte, maxBufferSize) // predefined content of zeroes
	index := 0

	data[index] = byte(value>>(intSize*8-7)) & 0b01000000 // set sign bit if i<0
	value = value ^ (value >> (intSize*8 - 1))            // if(i<0) i = ~i

	data[index] |= byte(value) & 0b00111111 // pack 6bit into data
	value >>= 6                             // discard 6 bits

	if value != 0 {
		data[index] |= 0b10000000 // set extend bit

		for {
			index++
			data[index] = byte(value) & 0b01111111 //  pack 7 bits
			value >>= 7                            // discard 7 bits

			if value != 0 {
				data[index] |= 1 << 7 // set extend bit
			} else {
				break // break if value is 0
			}

		}
	}

	index++
	data = data[:index] // ignore unused 'space'
	v.Compressed = append(v.Compressed, data...)
}
