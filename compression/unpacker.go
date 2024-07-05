package compression

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

const (
	Sanitize                SanitizeKind = 1
	SanitizeCC              SanitizeKind = 2
	SanitizeSkipWhitespaces SanitizeKind = 4
)

var (
	// ErrNoDataToUnpack is returned if the compressed array does not have sufficient data to unpack
	ErrNoDataToUnpack = fmt.Errorf("%w: no data", io.EOF)

	// ErrNotAString if no separator after a string is found, the string cannot be unpacked, as there is no string
	ErrNotAString = errors.New("could not unpack string: terminator not found")

	// ErrNotABool is returned when the data is neither 0x00 nor 0x01
	ErrNotABool = errors.New("could not unpack bool: invald value")

	// ErrNotEnoughData is used when the user tries to retrieve more data with NextBytes() than there is available.
	ErrNotEnoughData = errors.New("trying to read more data than available")
)

type SanitizeKind int

// NewUnpacker constructs a new Unpacker
func NewUnpacker(data []byte) *Unpacker {
	return &Unpacker{data}
}

// Unpacker unpacks received messages
type Unpacker struct {
	buffer []byte
}

// Reset resets the underlying byte slice to a new slice
// The slice that is passed to this method should not be used
// as the ownership has been passed to the unpacker.
func (u *Unpacker) Reset(b []byte) {
	u.buffer = b
}

// Size of the underlying buffer
func (u *Unpacker) Size() int {
	return len(u.buffer)
}

func (u *Unpacker) NextBool() (bool, error) {
	if len(u.buffer) == 0 {
		return false, ErrNoDataToUnpack
	}
	var b bool
	switch u.buffer[0] {
	case 0x00:
		b = false
	case 0x01:
		b = true
	default:
		return false, ErrNotABool
	}
	u.buffer = u.buffer[1:]
	return b, nil
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

// NextRawString unpacks the next string from the message without sanitizig it.
func (u *Unpacker) NextRawString() (s string, err error) {
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

// Bytes returns the not yet consumed bytes.
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

// first byte of the current buffer
func (u *Unpacker) peekByte(offset int) (byte, error) {
	if len(u.buffer) < offset+1 {
		return 0, ErrNotEnoughData
	}
	return u.buffer[offset], nil
}

func (u *Unpacker) NextSanitizedString(sanitizeType SanitizeKind) (string, error) {

	i := bytes.IndexByte(u.buffer, StringTerminator)
	if i < 0 {
		return "", ErrNotAString
	}

	var (
		// reduce slice reallocations by approximating the size
		// real size might be less due to sanitization
		result   = make([]byte, 0, i)
		skipping = sanitizeType&SanitizeSkipWhitespaces != 0
		err      error
		b        byte
		index    = -1
	)

	for {
		index++

		b, err = u.peekByte(index)
		if err != nil {
			return "", err
		}
		if b == StringTerminator {
			break
		}

		if skipping {
			if b == ' ' || b == '\t' || b == '\n' {
				continue
			}
			skipping = false
		}

		if sanitizeType&SanitizeCC != 0 {
			if b < 32 {
				b = ' '
			}
		} else if sanitizeType&Sanitize != 0 {
			if b < 32 && !(b == '\r') && !(b == '\n') && !(b == '\t') {
				b = ' '
			}
		}

		result = append(result, b)
	}

	u.buffer = u.buffer[index+1:]
	return string(result), nil
}

// NextString unpacks the next string from the message
// and sanitizes it by replacing control characters with spaces.
func (u *Unpacker) NextString() (string, error) {
	return u.NextSanitizedString(Sanitize)
}
