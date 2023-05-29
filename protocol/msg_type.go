package protocol

const (
	NetMsgNull MsgType = 0

	// the first thing sent by the client
	// contains the version info for the client
	NetMsgInfo MsgType = 1

	// sent by server
	NetMsgMapChange   MsgType = 2 // sent when client should switch map
	NetMsgMapData     MsgType = 3 // map transfer, contains a chunk of the map file
	NetMsgServerinfo  MsgType = 4
	NetMsgConReady    MsgType = 5  // connection is ready, client should send start info
	NetMsgSnap        MsgType = 6  // normal snapshot, multiple parts
	NetMsgSnapEmpty   MsgType = 7  // empty snapshot
	NetMsgSnapSingle  MsgType = 8  // ?
	NetMsgSnapSmall   MsgType = 9  //
	NetMsgInputTiming MsgType = 10 // reports how off the input was
	NetMsgRconAuthOn  MsgType = 11 // rcon authentication enabled
	NetMsgRconAuthOff MsgType = 12 // rcon authentication disabled
	NetMsgRconLine    MsgType = 13 // line that should be printed to the remote console
	NetMsgRconCmdAdd  MsgType = 14
	NetMsgRconCmdRem  MsgType = 15

	NetMsgAuthChallange MsgType = 16 //
	NetMsgAuthResult    MsgType = 17 //

	// sent by client
	NetMsgReady          MsgType = 18 //
	NetMsgEntergame      MsgType = 19
	NetMsgInput          MsgType = 20 // contains the inputdata from the client
	NetMsgRconCmd        MsgType = 21 //
	NetMsgRconAuth       MsgType = 22 //
	NetMsgRequestMapData MsgType = 23 //

	NetMsgAuthStart    MsgType = 24 //
	NetMsgAuthResponse MsgType = 25 //

	// sent by both
	NetMsgPing      MsgType = 26
	NetMsgPingReply MsgType = 27
	NetMsgError     MsgType = 28

	NetMsgMaplistEntryAdd MsgType = 29
	NetMsgMaplistEntryRem MsgType = 30
)

type MsgType int
