# Teeworlds API written in Go

## ![Test](https://github.com/jxsl13/twapi/workflows/Test/badge.svg) ![Go Report](https://goreportcard.com/badge/github.com/jxsl13/twapi) [![GoDoc](https://godoc.org/github.com/jxsl13/twapi?status.svg)](https://godoc.org/github.com/jxsl13/twapi) [![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT) [![codecov](https://codecov.io/gh/jxsl13/twapi/branch/master/graph/badge.svg)](https://codecov.io/gh/jxsl13/twapi) [![Total alerts](https://img.shields.io/lgtm/alerts/g/jxsl13/twapi.svg?logo=lgtm&logoWidth=18)](https://lgtm.com/projects/g/jxsl13/twapi/alerts/) [![codebeat badge](https://codebeat.co/badges/4b5339f2-93d6-4242-96a6-0372e66a7aaf)](https://codebeat.co/projects/github-com-jxsl13-twapi-master) [![Sourcegraph](https://sourcegraph.com/github.com/jxsl13/twapi/-/badge.svg)](https://sourcegraph.com/github.com/jxsl13/twapi?badge) [![deepsource](https://static.deepsource.io/deepsource-badge-light.svg)](https://deepsource.io/gh/jxsl13/twapi/)

Currently this supports only the server browser api.
It is possible to retrieve data from the masterservers as well as the server information from the game servers.

In order to download the latest released version, execute:

```shell
go get github.com/jxsl13/twapi@latest
```

In order to download the latest development version, execute:

```shell
go get github.com/jxsl13/twapi@master
```

## Stable packages

- browser
- compression
- config
- econ

## Unstable packages

- network
- protocol


### Example - High Level Abstraction(Open for optimizing suggestions)

```Go
package main

import (
    "fmt"
    "github.com/jxsl13/twapi/browser"
)

func main() {
    // fetch all server infos (that respond within 16 seconds)
    // if no servers responded, this list will be empty.
    infos := browser.ServerInfos()
    for _, info := range infos {
        fmt.Println(info)
    }

    // fetches the specified server's players, server name, etc.
    // if no answer is received within 16 seconds, this function returns
    // an error
    info, err := browser.GetServerInfo("89.163.148.121", 8305)
    if err != nil {
        fmt.Println(err)
    } else {
        fmt.Println(info)
    }
}
```

### Example - Slightly Higher Level Abstraction - Retrieve Serverlist

```Go
package main

import (
    "fmt"
    "net"
    "time"

    "github.com/jxsl13/twapi/browser"
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

    err = browser.RequestToken(conn)
    if err != nil {
        fmt.Println(err)
        return
    }

    conn.SetDeadline(time.Now().Add(5 * time.Second))

    buffer := [1500]byte{}
    bufSlice := buffer[:]

    read, err := conn.Read(bufSlice)
    if err != nil {
        fmt.Println(err)
        return
    }
    bufSlice = bufSlice[:read]
    fmt.Printf("read: %d bytes from %s\n", read, conn.RemoteAddr().String())

    // create toke from response
    token, err := browser.ParseToken(bufSlice)
    if err != nil {
        fmt.Println(err)
        return
    }
    // reset slice
    bufSlice = bufSlice[:1500]

    err = browser.Request("serverlist", token, conn)
    if err != nil {
        fmt.Println(err)
        return
    }

    // timeout after 5 secods
    // should not return an error
    conn.SetDeadline(time.Now().Add(5 * time.Second))

    // wait for response or time out
    read, err = conn.Read(bufSlice)
    bufSlice = bufSlice[:read]
    fmt.Printf("read: %d bytes from %s\n", read, conn.RemoteAddr().String())

    serverList, err := browser.ParseServerList(bufSlice)
    if err != nil {
        fmt.Println(err)
        return
    }

    for _, server := range serverList {
        fmt.Printf("Server: %s\n", server.String())
    }

}

```

### Example - Low Level - Retrieve Serverlist

```Go
package main

import (
    "fmt"
    "net"
    "time"

    "github.com/jxsl13/twapi/browser"
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
    fmt.Printf("read: %d bytes from %s\n", read, conn.RemoteAddr().String())

    // create toke from response
    token, err := browser.ParseToken(bufSlice)
    if err != nil {
        fmt.Println(err)
        return
    }
    // reset slice
    bufSlice = bufSlice[:1500]

    // create a new request from token
    serverListReq, err := browser.NewServerListRequestPacket(token)

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
    fmt.Printf("read: %d bytes from %s\n", read, conn.RemoteAddr().String())

    serverList, err := browser.ParseServerList(bufSlice)
    if err != nil {
        fmt.Println(err)
        return
    }

    for _, server := range serverList {
        fmt.Printf("Server: %s\n", server.String())
    }
}
```
