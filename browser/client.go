package browser

import (
	"net"
	"sync"
	"time"
)

func newClient() (*Client, error) {
	c := &Client{
		tokenCache: newTokenCache(),
	}

	var err error
	c.conn, err = net.ListenUDP("udp", nil)
	if err != nil {
		return nil, err
	}

	err = c.SetReadBuffer(maxBufferSize * maxChunks)
	if err != nil {
		defer c.conn.Close()
		return nil, err
	}

	err = c.SetWriteBuffer(maxBufferSize)
	if err != nil {
		defer c.conn.Close()
		return nil, err
	}

	c.SetReadTimeout(TokenExpirationDuration)
	c.SetWriteTimeout(100 * time.Millisecond)
	return c, nil
}

// NewClient creates a new browser client that can fetch the number of registered servers,
func NewClient(address string) (*Client, error) {
	c, err := newClient()
	if err != nil {
		return nil, err
	}
	err = c.SetTarget(address)
	if err != nil {
		defer c.Close()
		return nil, err
	}
	return c, nil
}

// Client is a browser client tha can fetch master server infos and server infos of game servers
type Client struct {
	conn         *net.UDPConn
	target       *net.UDPAddr
	readTimeout  time.Duration
	writeTimeout time.Duration
	tokenCache   *tokenCache

	mu sync.Mutex
}

func (c *Client) SetTarget(address string) error {

	addr, err := parseAddress(address)
	if err != nil {
		return err
	}

	c.setTarget(addr)
	return nil
}

func (c *Client) setTarget(address *net.UDPAddr) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.target = address
}

func (c *Client) SetReadTimeout(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.readTimeout = d
}

func (c *Client) SetWriteTimeout(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writeTimeout = d
}

func (c *Client) SetReadBuffer(bytes int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.SetReadBuffer(bytes)
}

func (c *Client) SetWriteBuffer(bytes int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.SetWriteBuffer(bytes)
}

func (c *Client) Close() error {
	return c.conn.Close()
}

// tries to write all of the data provided
func (c *Client) write(data []byte) (int, error) {
	return c.writeToUDP(data, c.target)
}

// tries to write all of the data provided
func (c *Client) writeToUDP(data []byte, addr *net.UDPAddr) (int, error) {
	expected := len(data)
	written := 0
	for written < expected {
		err := c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
		if err != nil {
			return -1, err
		}
		i, err := c.conn.WriteToUDP(data, addr)
		if err != nil {
			return written, err
		}
		written += i
	}
	return written, nil
}

func (c *Client) read(data []byte) (int, error) {
	err := c.conn.SetReadDeadline(time.Now().Add(c.readTimeout))
	if err != nil {
		return 0, err
	}
	return c.conn.Read(data)
}

func (c *Client) readFromUDP(data []byte) (int, *net.UDPAddr, error) {
	err := c.conn.SetReadDeadline(time.Now().Add(c.readTimeout))
	if err != nil {
		return 0, nil, err
	}
	return c.conn.ReadFromUDP(data)
}

// unguarded variant
func (c *Client) getToken() (*Token, error) {
	addr := c.target.String()
	// no need to refresh token if it has not yet expired
	if !c.tokenCache.Get(addr).Expired() {
		return c.tokenCache.Get(addr), nil
	}
	// TODO: check if we need to process the number of written bytes
	_, err := c.write(NewTokenRequestPacket())
	if err != nil {
		return nil, err
	}
	buffer := [maxBufferSize]byte{}
	response := buffer[:]
	i, err := c.read(response)
	if err != nil {
		return nil, err
	}
	response = response[:i] // shorten slice to only contain read bytes

	token := &Token{}
	err = token.UnmarshalBinary(response)
	if err != nil {
		return nil, err
	}

	c.tokenCache.Add(addr, token)
	return token, nil
}

// GetToken returns a client/server token that secures the connection against IP spoofing
func (c *Client) GetToken() (*Token, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.getToken()
}

// request creates the request payload
func (c *Client) request(header string) ([]byte, error) {
	// refresh token if not expired
	token, err := c.getToken()
	if err != nil {
		return nil, err
	}
	req := RequestPacket{
		Token:  *token,
		Header: header,
	}
	body, err := req.MarshalBinary()
	if err != nil {
		return nil, err
	}

	return body, nil
}

// send request and receive a single chunk with the response
func (c *Client) get(header string) (*ResponsePacket, error) {
	request, err := c.request(header)
	if err != nil {
		return nil, err
	}
	_, err = c.write(request)
	if err != nil {
		return nil, err
	}
	buffer := [maxBufferSize]byte{}
	response := buffer[:]
	i, err := c.read(response)
	if err != nil {
		return nil, err
	}
	data := response[:i]

	resp := &ResponsePacket{}
	err = resp.UnmarshalBinary(data)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetServerCount returns the number of registered servers for the current master server
func (c *Client) GetServerCount() (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.getServerCount()
}

func (c *Client) getServerCount() (int, error) {
	resp, err := c.get(RequestServerCount)
	if err != nil {
		return -1, err
	}
	i, err := parseServerCount(resp.Payload)
	if err != nil {
		return -1, err
	}
	return i, nil
}

// GetServerAddresses returns a list of server addresses from the underlying master server
func (c *Client) GetServerAddresses() ([]*net.UDPAddr, error) {
	c.mu.Lock()
	list, err := c.getServerAddresses()
	c.mu.Unlock()
	if err != nil {
		return nil, err
	}
	// cleanup duplicate, most likely ipv4&ipv6 addresses
	set := make(map[string]*net.UDPAddr, len(list))
	for _, addr := range list {
		set[addr.String()] = addr
	}
	list = list[:0] // reset list
	for _, addr := range set {
		list = append(list, addr)
	}
	return list, nil
}

func (c *Client) getServerAddresses() ([]*net.UDPAddr, error) {
	expectedServers, err := c.getServerCount()
	if err != nil {
		return nil, err
	}
	expectedChunks := expectedServers / maxServersPerChunk
	if expectedServers%maxServersPerChunk > 0 {
		expectedChunks += 1
	}

	result := make([]*net.UDPAddr, 0, expectedServers)

	request, err := c.request(RequestServerList)
	if err != nil {
		return nil, err
	}
	_, err = c.write(request)
	if err != nil {
		return nil, err
	}
	// we expect 75 addresses per chunk, thus we expect multiple chunks of data
	// that we wanna parse
	buffer := [maxBufferSize]byte{}
	for i := 0; i < expectedChunks; i++ {
		response := buffer[:]
		i, err := c.read(response)
		if err != nil {
			return nil, err
		}
		data := response[:i]

		resp := ResponsePacket{}
		err = resp.UnmarshalBinary(data)
		if err != nil {
			return nil, err
		}

		list, err := parseServerList(resp.Payload)
		if err != nil {
			return nil, err
		}
		result = append(result, list...)
	}

	return result, nil
}

func (c *Client) getServerInfo() (*ServerInfo, error) {
	resp, err := c.get(RequestInfo)
	if err != nil {
		return nil, err
	}

	info, err := parseServerInfo(resp.Payload, c.target.String())
	if err != nil {
		return nil, err
	}

	return info, nil
}

// GetServerInfo returns the server info of a game server. This function requires the target to be set to a
// game server address
func (c *Client) GetServerInfo() (*ServerInfo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.getServerInfo()
}
