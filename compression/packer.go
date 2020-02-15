package compression

// Packer compresses data
type Packer struct {
	Buffer []byte
}

// init initializes the buffer if it's nil
func (p *Packer) init() {
	if p.Buffer == nil {
		p.Buffer = make([]byte, 0, packerInitialSize)
	}
}

// Bytes returns the underlying buffer
func (p *Packer) Bytes() []byte {
	p.init()

	return p.Buffer
}

// Reset Buffer
func (p *Packer) Reset() {
	p.init()
	p.Buffer = p.Buffer[:0]
}

// Size len of the Buffer
func (p *Packer) Size() int {
	return len(p.Buffer)
}

// Add integer, bytes or string
func (p *Packer) Add(data interface{}) {
	p.init()

	switch data.(type) {
	case int:
		var v VarInt
		v.Pack(data.(int))
		p.Buffer = append(p.Buffer, v.Bytes()...)

	case string:
		p.Buffer = append(p.Buffer, []byte(data.(string))...)
		p.Buffer = append(p.Buffer, byte(0)) // string separator

	case []byte:
		p.Buffer = append(p.Buffer, data.([]byte)...)

	default:
		panic(ErrTypeNotSupported)
	}

	return
}

// Unpacker unpacks received messages
type Unpacker struct {
	Buffer []byte
}

// Reset resets the underlying byte slice to a new slice
func (u *Unpacker) Reset(b []byte) {
	u.Buffer = b
}

// Size of the underlying buffer
func (u *Unpacker) Size() int {
	return len(u.Buffer)
}

// NextInt unpacks the next integer
func (u *Unpacker) NextInt() (i int, err error) {
	v := VarInt{u.Buffer}
	i, err = v.Unpack()
	u.Buffer = v.Bytes()
	return
}

// NextString unpacks the next string from the message
func (u *Unpacker) NextString() (s string, err error) {
	if len(u.Buffer) == 0 {
		err = ErrNoDataToUnpack
		return
	}

	foundSeparator := false
	separatorPos := 0
	for idx, b := range u.Buffer {
		if b == 0 {
			foundSeparator = true
			separatorPos = idx
			break
		}
	}

	if !foundSeparator {
		err = ErrNoStringToUnpack
		return
	}

	s = string(u.Buffer[:separatorPos])
	u.Buffer = u.Buffer[separatorPos+1:] // skip separator
	return
}

// NextBytes returns the next size bytes.
func (u *Unpacker) NextBytes(size int) (b []byte, err error) {
	if len(u.Buffer) < size || size < 0 {
		err = ErrNotEnoughDataToUnpack
		return
	}

	b = u.Buffer[:size]
	u.Buffer = u.Buffer[size:]
	return
}
