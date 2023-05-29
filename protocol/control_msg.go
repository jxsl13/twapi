package protocol

const (
	NetCtrlMsgKeepAlive ControlMsg = 0
	NetCtrlMsgConnect   ControlMsg = 1
	NetCtrlMsgAccept    ControlMsg = 2
	NetCtrlMsgClose     ControlMsg = 4
	NetCtrlMsgToken     ControlMsg = 5
)

type ControlMsg int
