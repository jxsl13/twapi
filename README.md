# Teeworlds API written in Go
Currently this supports only the server browser api.
It is possible to retrieve data from the masterservers as well as the server information from the game servers.


### Example - Retrieve Serverlist
```Go
package main

import (
	"fmt"
	"github.com/jxsl13/twapi/browser"
	"net"
	"time"
)

func main() {
	addr, err := net.ResolveUDPAddr("udp", "master1.teeworlds.com:8283")
	if err != nil {
		fmt.Println(err)
		return
	}

	conn, err := net.DialUDP("udp", nil, addr)

	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()

	// token request packet
	tokenReq := browser.NewTokenRequestPacket()

	// send request
	written, err := conn.Write(tokenReq)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("written: %d bytes to %s\n", written, conn.RemoteAddr().String())

	conn.SetDeadline(time.Now().Add(5 * time.Second))

	buffer := [1500]byte{}
	bufSlice := buffer[:]

	read, err := conn.Read(bufSlice)
	if err != nil {
		fmt.Println(err)
		return
	}
	bufSlice = bufSlice[:read]
	fmt.Printf("read: %d bytes from %s\n", written, conn.RemoteAddr().String())

	// create toke from response
	token, err := browser.ParseToken(bufSlice)
	// reset slice
	bufSlice = bufSlice[:1500]

	// create a new request from token
	serverListReq, err := browser.ParseServerListRequestPacket(token)

	// Send server list request
	written, err = conn.Write(serverListReq)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("written: %d bytes to %s\n", written, conn.RemoteAddr().String())

	// timeout after 5 secods
	// should not return an error
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// wait for response or time out
	read, err = conn.Read(bufSlice)
	bufSlice = bufSlice[:read]
	fmt.Printf("read: %d bytes from %s\n", written, conn.RemoteAddr().String())

	serverList, err := browser.NewServerList(bufSlice)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, server := range serverList {
		fmt.Printf("Server: %s\n", server.String())
	}

}

```
