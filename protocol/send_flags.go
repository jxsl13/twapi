package protocol

const (
	NetSendFlagVital    SendFlags = 1
	NetSendFlagConnless SendFlags = 2
	NetSendFlagFlush    SendFlags = 4
)

type SendFlags int
