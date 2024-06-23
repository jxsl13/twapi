package compression

const (
	// StringTerminator is the zero byte that terminates the string
	StringTerminator byte = 0

	// with how many bytes the packer is initialized
	PackerBufferSize = 1024 * 2
)

// NewPacker ceates a new Packer with a given default buffer size.
// You can provide ONE optional buffer that is used instead of the default one
func NewPacker(buf ...[]byte) *Packer {
	var internalBuf []byte
	if len(buf) > 0 {
		internalBuf = buf[0]
	} else {
		internalBuf = make([]byte, 0, PackerBufferSize)
	}

	return &Packer{
		buffer: internalBuf,
	}
}

// Packer compresses data
type Packer struct {
	buffer []byte
}

// Bytes returns the underlying buffer
func (p *Packer) Bytes() []byte {
	return p.buffer
}

// Reset internal buffer
func (p *Packer) Reset(buf ...[]byte) {
	if len(buf) > 0 {
		p.buffer = append(p.buffer[:0], buf[0]...)
	} else if p.buffer != nil {
		p.buffer = p.buffer[:0]
	} else {
		p.buffer = make([]byte, 0, PackerBufferSize)
	}
}

// Size len of the Buffer
func (p *Packer) Size() int {
	return len(p.buffer)
}

func (p *Packer) AddInt(i int) {
	p.buffer = AppendVarint(p.buffer, i)
}

func (p *Packer) AddByte(b byte) {
	p.buffer = append(p.buffer, b)
}

func (p *Packer) AddString(s string) {
	p.buffer = append(p.buffer, []byte(s)...)
	p.buffer = append(p.buffer, StringTerminator) // string separator
}

func (p *Packer) AddBytes(data []byte) {
	p.buffer = append(p.buffer, data...)
}
