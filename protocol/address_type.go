package protocol

const (
	NetTypeInvalid       NetType = 0
	NetTypeIPv4          NetType = 1
	NetTypeIPv6          NetType = 2
	NetTypeLinkBroadcast NetType = 4
)

type NetType int
