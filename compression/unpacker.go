package compression

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

var (
	// ErrNoDataToUnpack is returned if the compressed array does not have sufficient data to unpack
	ErrNoDataToUnpack = fmt.Errorf("%w: no data", io.EOF)

	// ErrNotAString if no separator after a string is found, the string cannot be unpacked, as there is no string
	ErrNotAString = errors.New("could not unpack string: terminator not found")

	// ErrNotEnoughData is used when the user tries to retrieve more data with NextBytes() than there is available.
	ErrNotEnoughData = errors.New("trying to read more data than available")
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
		return 0, ErrNoDataToUnpack
	} else if n < 0 {
		return i, fmt.Errorf("invalid varint data: overflow %d", n)
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
		return "", ErrNotAString
	}

	s = string(u.buffer[:i])
	u.buffer = u.buffer[i+1:] // skip separator
	return
}

// NextBytes returns the next size bytes.
func (u *Unpacker) NextBytes(size int) (b []byte, err error) {
	if size < 0 {
		panic("negative size")
	} else if len(u.buffer) < size {
		return nil, fmt.Errorf("%w: requesting %d, got %d", ErrNotEnoughData, size, len(u.buffer))
	}

	result := make([]byte, size)
	copy(result, u.buffer[:size])
	u.buffer = u.buffer[size:]
	return result, nil
}

// NextByte returns the next bytes.
func (u *Unpacker) NextByte() (b byte, err error) {
	if len(u.buffer) == 0 {
		return 0, ErrNotEnoughData
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
