package main

import (
	"fmt"
	"sync"
	"time"
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
	cache := NewConcurrentServerCache(2048)

	ms, _ := NewMasterServerFromAddress("master1.teeworlds.com:8283")
	srvAddresses, _ := ms.GetServerList()

	fmt.Printf("Retrieved %d server addresses from master server.\n", len(srvAddresses))
	gameservers := make([]GameServer, 0, len(srvAddresses)*2)
	for _, addr := range srvAddresses {
		srv, _ := NewGameServerFromUDPAddr(&addr, 5*time.Second, 5)

		gameservers = append(gameservers, srv)
	}

	start := func(gs GameServer, c *ConcurrentServerCache, wg *sync.WaitGroup) {
		defer wg.Done()
		info, err := gs.ServerInfo()
		if err != nil {
			fmt.Print(err)
			return
		}
		if info.Valid() {
			cache.Add(&info)
		}
		fmt.Printf("Retrieving data from: %s\n", gs.RemoteAddr().String())
	}

	wg := sync.WaitGroup{}

	lim := 5
	for _, gs := range gameservers {
		lim--
		if lim == 0 {
			break
		}
		if gs.ValidConnection() {
			wg.Add(1)
			go start(gs, cache, &wg)
		}
	}

	wg.Wait()

}
