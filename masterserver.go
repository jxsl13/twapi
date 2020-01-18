package main

import "net"

// MasterServer represents a masterserver
type MasterServer struct {
	IP   net.IP
	Port int
}

// NewMasterServer creates a new masterserver
func NewMasterServer(host string, port int) (ms MasterServer, err error) {
	ips, err := net.LookupHost(host)
	if err != nil {
		return
	}
	ms = MasterServer{
		IP:   net.ParseIP(ips[0]),
		Port: port,
	}
	return
}

// GetUDPAddress returns the masterserver's address
func (ms *MasterServer) GetUDPAddress() net.UDPAddr {

	return net.UDPAddr{
		IP:   ms.IP,
		Port: ms.Port,
	}
}
