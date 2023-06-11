package network

import (
	"context"
	"fmt"
	"net/netip"
)

type NetClient struct {
	base *NetBase

	conn         Conn
	recvUnpacker NetRecvUnpacker
	tokenManager *TokenManager
	tokenCache   TokenCache
}

func NewNetClient(ctx context.Context, bindAddr netip.AddrPort, randomPort bool) (*NetClient, error) {
	sock, err := NewNetSocket(bindAddr, randomPort)
	if err != nil {
		return nil, fmt.Errorf("failed to create net client: %w", err)
	}
	base, err := NewNetBase(ctx, sock)
	if err != nil {
		return nil, fmt.Errorf("failed to create net client: %w", err)
	}

	var (
		tokenManager = NewTokenManager()
		tokenCache   = NewTokenCache(tokenManager) // TODO: continue here
	)

	return &NetClient{
		base:         base,
		conn:         NewConn(base, true),
		recvUnpacker: NewRecvUnpacker(),
		tokenManager: tokenManager,
		tokenCache:   tokenCache,
	}, nil
}

/*
class CNetClient : public CNetBase
{
	CNetConnection m_Connection;
	CNetRecvUnpacker m_RecvUnpacker;

	CNetTokenCache m_TokenCache;
	CNetTokenManager m_TokenManager;

	int m_Flags;

public:
	// openness
	bool Open(NETADDR BindAddr, class CConfig *pConfig, class IConsole *pConsole, class IEngine *pEngine, int Flags);
	void Close();

	// connection state
	int Disconnect(const char *Reason);
	int Connect(NETADDR *Addr);

	// communication
	int Recv(CNetChunk *pChunk, TOKEN *pResponseToken = 0);
	int Send(CNetChunk *pChunk, TOKEN Token = NET_TOKEN_NONE, CSendCBData *pCallbackData = 0);
	void PurgeStoredPacket(int TrackID);

	// pumping
	int Update();
	int Flush();

	int ResetErrorString();

	// error and state
	int State() const;
	bool GotProblems() const;
	const char *ErrorString() const;
};
*/
