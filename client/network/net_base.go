package network

import "net"

var (
	netInitializer = NewNetInitializer()
)

type NetInitializer struct {
}

func NewNetInitializer() NetInitializer {
	return NetInitializer{}
}

type NetBase struct {
	Socket net.UDPConn
}

// TODO: Continue here when huffman is ready.
/**
class CNetBase
{

	NETSOCKET m_Socket;
	IOHANDLE m_DataLogSent;
	IOHANDLE m_DataLogRecv;
	CHuffman m_Huffman;
	unsigned char m_aRequestTokenBuf[NET_TOKENREQUEST_DATASIZE];

public:
	CNetBase();
	~CNetBase();
	CConfig *Config() { return m_pConfig; }
	class IEngine *Engine() { return m_pEngine; }
	int NetType() { return m_Socket.type; }

	void Init(NETSOCKET Socket, class CConfig *pConfig, class IConsole *pConsole, class IEngine *pEngine);
	void Shutdown();
	void UpdateLogHandles();
	void Wait(int Time);

	void SendControlMsg(const NETADDR *pAddr, TOKEN Token, int Ack, int ControlMsg, const void *pExtra, int ExtraSize);
	void SendControlMsgWithToken(const NETADDR *pAddr, TOKEN Token, int Ack, int ControlMsg, TOKEN MyToken, bool Extended);
	void SendPacketConnless(const NETADDR *pAddr, TOKEN Token, TOKEN ResponseToken, const void *pData, int DataSize);
	void SendPacket(const NETADDR *pAddr, CNetPacketConstruct *pPacket);
	int UnpackPacket(NETADDR *pAddr, unsigned char *pBuffer, CNetPacketConstruct *pPacket);
};

*/
