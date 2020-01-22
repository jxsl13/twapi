package main

import (
	"fmt"
	"log"
	"net"
)

func getMasterServerList() (masterservers []*MasterServer) {
	for i := 1; i <= 4; i++ {
		msAddress := fmt.Sprintf("master%d.teeworlds.com:%d", i, 8283)
		ms, err := NewMasterServerFromAddress(msAddress)
		if err != nil {
			continue
		}

		masterservers = append(masterservers, &ms)
	}
	return
}

func main() {
	masterServers := getMasterServerList()
	servers := make([]net.UDPAddr, 0, 1024)

	for _, ms := range masterServers {
		defer ms.Close()
	}

	for _, ms := range masterServers {
		list, err := ms.GetServerList()
		if err != nil {
			log.Printf("error retrieving server list from masterserver %s: %s", ms.UDPConn.RemoteAddr(), err)
		} else {
			servers = append(servers, list...)
		}
	}

	for _, srv := range servers {
		fmt.Println(srv)
	}

}
