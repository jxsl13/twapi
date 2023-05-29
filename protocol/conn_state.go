package protocol

const (
	ConnStateOffline ConnState = 0
	ConnStateToken   ConnState = 1
	ConnStateConnect ConnState = 2
	ConnStatePending ConnState = 3
	ConnStateOnline  ConnState = 4
	ConnStateError   ConnState = 5
)

type ConnState int
