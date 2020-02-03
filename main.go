package main

import (
	"fmt"
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

}
