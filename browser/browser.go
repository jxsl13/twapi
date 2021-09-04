package browser

import (
	"errors"
	"log"
	"net"
	"sync"
	"time"
)

const (
	// Used for master server
	// RequestServerList is a header used in order to fetch the server list from the master server
	RequestServerList = "\xff\xff\xff\xffreq2"
	// RequestServerCount is a header used in order to fetch the number of servers from the master server
	RequestServerCount = "\xff\xff\xff\xffcou2"

	// SendServerList is used by the master server when sending the server list
	SendServerList = "\xff\xff\xff\xfflis2"
	// SendServerCount is used by the master server when sending the server count
	SendServerCount = "\xff\xff\xff\xffsiz2"

	// Used for the gameserver
	// RequestInfo is used in order to request the server info of a game server
	RequestInfo = "\xff\xff\xff\xffgie3\x00" // need explicitly the trailing \x00
	// SendInfo is used
	SendInfo = "\xff\xff\xff\xffinf3\x00"

	// Packet versioning for packing token header
	netPacketFlagConnless = 8
	netPacketVersion      = 1

	minHeaderLength = 8 // length of the shortest header
	maxHeaderLength = 9 // length of the longest header

	tokenResponseSize = 12                                       // size of the token that is sent after the TokenRequest
	tokenPrefixSize   = netPacketFlagConnless + netPacketVersion // size of the token that is sent as prefix to every follow up message.

	minPrefixLength = tokenResponseSize
	maxPrefixLength = tokenPrefixSize + maxHeaderLength

	maxBufferSize      = 1500
	maxChunks          = 16
	maxServersPerChunk = 75
)

var (
	// binary delimiter
	delimiter = []byte("\x00")

	// Logging can be set to "true" in order to see more logging output from the package.
	Logging = false

	// TimeoutMasterServers is used by ServerInfos as a value that drops few packets
	TimeoutMasterServers = 5 * time.Second

	// TimeoutServers is also used by ServerInfos as a alue that drops few packets
	TimeoutServers = TokenExpirationDuration

	// ErrInvalidIP is returned if the passed IP to some function is not a valid IP address
	ErrInvalidAddress = errors.New("invalid address passed")

	// ErrInvalidIP is returned if the passed IP to some function is not a valid IP address
	ErrInvalidIP = errors.New("invalid IP passed")

	// ErrInvalidPort is returned if the passed port is either negative or an invalid value above 65536.
	ErrInvalidPort = errors.New("invalid port passed")

	// ErrTokenExpired is returned when a request packet is being constructed with an expired token
	ErrTokenExpired = errors.New("token expired")

	// ErrInvalidResponseMessage is returned when a passed response message does not contain the expected data.
	ErrInvalidResponseMessage = errors.New("invalid response message")

	// ErrInvalidHeaderLength is returned, if a too short byte slice is passed to some of the parsing methods
	ErrInvalidHeaderLength = errors.New("invalid header length")

	// ErrInvalidHeaderFlags is returned, if the first byte of a response message does not corespond to the expected flags.
	ErrInvalidHeaderFlags = errors.New("invalid header flags")

	// ErrUnexpectedResponseHeader is returned, if a message is passed to a parsing function, that expects a different response
	ErrUnexpectedResponseHeader = errors.New("unexpected response header")

	// ErrMalformedResponseData is reurned is a server sends broken or malformed response data
	// that cannot be properly parsed.
	ErrMalformedResponseData = errors.New("malformed response data")

	// ErrTimeout is used in Retry functions that support a timeout parameter
	ErrTimeout = errors.New("timeout")

	// ErrInvalidWrite is returned if writing to an io.Writer failed
	ErrInvalidWrite = errors.New("invalid write")

	// ErrRequestResponseMismatch is returned by functions that request and receive data, but the received data does not match the requested data.
	ErrRequestResponseMismatch = errors.New("request response mismatch")

	// TokenExpirationDuration sets the protocol expiration time of a token
	// This variable can be changed
	TokenExpirationDuration = time.Second * 16

	masterServerHostnameAddresses = []string{
		"master1.teeworlds.com:8283",
		"master2.teeworlds.com:8283",
		"master3.teeworlds.com:8283",
		"master4.teeworlds.com:8283",
	}

	// MasterServerAddresses contains the resolved addresses as ip:port
	// initialized on startup with master servers that can be reached
	MasterServerAddresses = []*net.UDPAddr{}

	// ResponsePacketList is a list of known headers that we can expect from either master or game servers
	ResponsePacketList = [][]byte{
		[]byte(SendServerCount),
		[]byte(SendServerList),
		[]byte(SendInfo),
	}
)

// init initializes a package on import
func init() {
	if Logging {
		log.Println("Initializing twapi package...")
	}

	MasterServerAddresses = make([]*net.UDPAddr, 0, len(masterServerHostnameAddresses))

	for _, ms := range masterServerHostnameAddresses {
		srv, err := net.ResolveUDPAddr("udp", ms)
		if err != nil {
			if Logging {
				log.Printf("Failed to resolve: %s\n", ms)
			}
		} else {
			if Logging {
				log.Printf("Resolved masterserver: %s -> %s\n", ms, srv.String())
			}
			MasterServerAddresses = append(MasterServerAddresses, srv)
		}
	}
	if Logging && len(MasterServerAddresses) == 0 {
		log.Println("Could not resolve any masterservers.... please check your internet connection.")
	}
}

func GetServerAddresses() ([]*net.UDPAddr, error) {
	clients := make([]*Client, len(MasterServerAddresses))
	for idx := range clients {
		client, err := NewClient(MasterServerAddresses[idx].String())
		if err != nil {
			return nil, err
		}
		// called at the end of the function, not at the end of the loop
		defer client.Close()
		clients[idx] = client
	}

	set := make(map[string]*net.UDPAddr, 1024)
	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}

	wg.Add(len(clients))
	for _, client := range clients {
		go func(c *Client, s map[string]*net.UDPAddr, m *sync.Mutex, w *sync.WaitGroup) {
			defer wg.Done()
			// get all addresses from all master servers
			list, err := c.GetServerAddresses()
			if err != nil {
				return
			}
			m.Lock()
			defer m.Unlock()
			// add all addresses to a global set
			for _, addr := range list {
				s[addr.String()] = addr
			}
		}(client, set, mu, wg)
	}

	wg.Wait()
	result := make([]*net.UDPAddr, 0, len(set))
	for _, addr := range set {
		result = append(result, addr)
	}
	return result, nil
}

// GetServerInfos returns a list of all server infos that are registered with the master servers
func GetServerInfosOf(addr []*net.UDPAddr) ([]*ServerInfo, error) {
	list := addr
	// channel with addresses that need to be fetched
	addresses := make(chan *net.UDPAddr)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	// routine that creates work, producer
	go func(w *sync.WaitGroup) {
		defer wg.Done()
		// push work into channel
		for _, addr := range list {
			addresses <- addr
		}
		// wait for worker threads to finish
		close(addresses)
	}(wg)

	// use global value
	workers := len(list)
	var gErr error
	mu := &sync.Mutex{}
	result := make([]*ServerInfo, 0, len(list))

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		// start worker routines
		go func(w *sync.WaitGroup, m *sync.Mutex, id int) {
			defer w.Done()
			if Logging {
				log.Printf("starting worker %d\n", id)
			}
			// one client per worker
			c, err := newClient()
			if err != nil {
				gErr = err
				return
			}
			defer c.Close()
			// iterate over channel until closed
			for addr := range addresses {
				// communicate with target address
				if Logging {
					log.Printf("fetching server info for address: %s\n", addr)
				}
				c.setTarget(addr)
				si, err := c.getServerInfo()
				if err != nil {
					continue
				}
				m.Lock()
				result = append(result, si)
				m.Unlock()
			}
		}(wg, mu, i)
	}

	// wait for workers to finish work
	wg.Wait()
	if gErr != nil {
		return nil, gErr
	}
	return result, nil
}

func missing(all []*net.UDPAddr, found map[string]*ServerInfo) []*net.UDPAddr {
	result := make([]*net.UDPAddr, 0, len(all)/2)
	missing := make(map[string]*net.UDPAddr, len(found)/2)

	for _, addr := range all {
		addrStr := addr.String()
		_, ok := found[addrStr]
		if !ok {
			missing[addrStr] = addr
		}
	}

	for _, mis := range missing {
		result = append(result, mis)
	}
	log.Printf("missing: %d\n", len(missing))
	return result
}

func GetServerInfos() ([]*ServerInfo, error) {
	list, err := GetServerAddresses()
	if err != nil {
		return nil, err
	}
	checkList := make(map[string]*ServerInfo, len(list))

	retries := 0
	for mis := missing(list, checkList); len(mis) > 0 && retries < 2; mis = missing(list, checkList) {
		retries++

		infos, err := GetServerInfosOf(mis)
		if err != nil {
			continue
		}

		for _, info := range infos {
			checkList[info.Address] = info
		}
	}

	result := make([]*ServerInfo, 0, len(checkList))
	for _, info := range checkList {
		result = append(result, info)

	}
	return result, nil

}
