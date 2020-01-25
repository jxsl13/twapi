package main

// VarInt is used to compress integers in a variable length format.
// Format: ESDDDDDD EDDDDDDD EDD... Extended, Data, Sign
// E: is next byte part of the current integer
// S: Sign of integer
// Data, Integer bits that follow the sign
type VarInt struct {
	compressed []byte
}

// NewVarIntFrom Bytes allows for a creation if a buffer based on a preexisting buffer
func NewVarIntFrom(bytes []byte) VarInt {
	return VarInt{bytes}
}

// Size returns the length of the data.
// not its capacity
func (v *VarInt) Size() int {
	if v.compressed == nil {
		v.Clear()
	}
	return len(v.compressed)
}

// Data returns the compressed data slice.
func (v *VarInt) Data() []byte {
	if v.compressed == nil {
		v.Clear()
	}
	return v.compressed
}

// Clear clears the internal compressed buffer
func (v *VarInt) Clear() {
	v.compressed = make([]byte, 0, 5)
}

// Unpack the wrapped compressed buffer
func (v *VarInt) Unpack() (value int) {
	if v.compressed == nil {
		v.Clear()
	}

	if len(v.compressed) == 0 {
		panic("Invalid usage, please check the size before trying to unpack")
	}

	index := 0

	data := v.compressed

	sign := (data[index] >> 6) & 0b00000001

	value = int(data[index] & 0b00111111)

	for {

		if data[index]&0b10000000 == 0 {
			break
		}
		index++
		value |= int(data[index]&0b01111111) << (6)

		if data[index]&0b10000000 == 0 {
			break
		}
		index++
		value |= int(data[index]&0b01111111) << (6 + 7)

		if data[index]&0b10000000 == 0 {
			break
		}
		index++
		value |= int(data[index]&0b01111111) << (6 + 7 + 7)

		if data[index]&0b10000000 == 0 {
			break
		}
		index++
		value |= int(data[index]&0b01111111) << (6 + 7 + 7 + 7)

		// break free in any case
		break
	}
	index++

	value ^= -int(sign) // if(sign) value = ~(value)

	// continue walking over the buffer
	v.compressed = v.compressed[index:]
	return
}

// Pack a value to internal buffer
func (v *VarInt) Pack(value int) {
	if v.compressed == nil {
		v.Clear()
	}

	data := make([]byte, 5)
	index := 0

	data[index] = byte(value>>25) & 0b01000000 // set sign bit if i<0
	value = value ^ (value >> 31)              // if(i<0) i = ~i

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
	data = data[:index] // discard unused 'space'
	v.compressed = append(v.compressed, data...)
}
