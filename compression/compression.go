package compression

import "errors"

var (
	// ErrNoDataToUnpack is returned if the compressed array does not have sufficient data to unpack
	ErrNoDataToUnpack = errors.New("no data")

	// ErrTypeNotSupported is returned, when an invalid type is being tried to be packed.
	ErrTypeNotSupported = errors.New("error: type not supported")

	// ErrNoStringToUnpack if no separator after a string is found, the string cannot be unpacked, as there is no string
	ErrNoStringToUnpack = errors.New("could not unpack string, as there is no separator to be found")

	// ErrNotEnoughDataToUnpack is used when the user tries to retrieve more data with NextBytes() than there is available.
	ErrNotEnoughDataToUnpack = errors.New("you are trying to read more data than is available")
)

const (
	// max bytes that can be received for one integer
	maxBytesInVarInt = 5

	// with how many bytes the packer is initialized
	packerInitialSize = 2048
)
