package compression

import (
	"bytes"
	"errors"
	"fmt"
)

var (
	// ErrNoDataToUnpack is returned if the compressed array does not have sufficient data to unpack
	ErrNoDataToUnpack = errors.New("no data")

	// ErrNoStringToUnpack if no separator after a string is found, the string cannot be unpacked, as there is no string
	ErrNoStringToUnpack = errors.New("could not unpack string: terminator not found")

	// ErrNotEnoughDataToUnpack is used when the user tries to retrieve more data with NextBytes() than there is available.
	ErrNotEnoughDataToUnpack = errors.New("trying to read more data than available")
)

// NewUnpacker constructs a new Unpacker
func NewUnpacker(data []byte) *Unpacker {
	return &Unpacker{data}
}

// Unpacker unpacks received messages
type Unpacker struct {
	buffer []byte
}

// Reset resets the underlying byte slice to a new slice
func (u *Unpacker) Reset(b []byte) {
	u.buffer = b
}

// Size of the underlying buffer
func (u *Unpacker) Size() int {
	return len(u.buffer)
}

// NextInt unpacks the next integer
func (u *Unpacker) NextInt() (i int, err error) {
	i, n := Varint(u.buffer)
	if n == 0 {
		return 0, errors.New("invalid data: no data")
	} else if n < 0 {
		return i, fmt.Errorf("invalid data: overflow %d", n)
	}
	u.buffer = u.buffer[n:]
	return i, nil
}

// NextString unpacks the next string from the message
func (u *Unpacker) NextString() (s string, err error) {
	if len(u.buffer) == 0 {
		return "", ErrNoDataToUnpack
	}

	i := bytes.IndexByte(u.buffer, StringTerminator)
	if i < 0 {
		return "", ErrNoStringToUnpack
	}

	s = string(u.buffer[:i])
	u.buffer = u.buffer[i+1:] // skip separator
	return
}

// NextBytes returns the next size bytes.
func (u *Unpacker) NextBytes(size int) (b []byte, err error) {
	if len(u.buffer) < size || size < 0 {
		return nil, ErrNotEnoughDataToUnpack
	}

	result := make([]byte, size)
	copy(result, u.buffer[:size])
	u.buffer = u.buffer[size:]
	return result, nil
}

// NextByte returns the next bytes.
func (u *Unpacker) NextByte() (b byte, err error) {
	if len(u.buffer) < 1 {
		return 0, ErrNotEnoughDataToUnpack
	}

	result := u.buffer[0]
	u.buffer = u.buffer[1:]
	return result, nil
}

// Bytes returns the not yet used bytes.
// This operation consumes the buffer leaving it empty
func (u *Unpacker) Bytes() []byte {
	if len(u.buffer) == 0 {
		return []byte{}
	}

	result := make([]byte, len(u.buffer))
	copy(result, u.buffer)
	u.buffer = u.buffer[:0]
	return result
}
