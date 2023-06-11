package protocol

const (
	// NetMaxSequence is the first invalid sequence number
	// it is used as modulo operand.
	NetMaxSequence  = 1 << 10
	NetSequenceMask = NetMaxSequence - 1
)
