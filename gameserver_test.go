package main

import (
	"bytes"
	"encoding/hex"
	"net"
	"testing"
	"time"
)

func TestServerInfo(t *testing.T) {

	udpAddr, err := net.ResolveUDPAddr("udp", highlyFrequentedServer)
	if err != nil {
		t.Error(err)
		return
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)

	if err != nil {
		t.Error(err)
		return
	}

	bc, err := NewBrowserConnetion(conn, 30*time.Second)
	if err != nil {
		t.Error(err)
		return
	}
	defer bc.Close()

	resp, err := bc.Request(getInfo, receiveInfo, 2048)
	if err != nil {
		t.Error(err)
		return
	}

	info := ServerInfo{}
	t.Logf("\nAddress: %s DataLength(%d)\n%s\n", conn.RemoteAddr(), len(resp), hex.Dump(resp))

	slots := bytes.SplitN(resp, []byte("\x00"), 6)

	info.Address = conn.RemoteAddr().String()
	info.Version = string(slots[0])
	info.Name = string(slots[1])
	info.Hostname = string(slots[2])
	info.Map = string(slots[3])
	info.GameType = string(slots[4])

	data := slots[5]

	info.ServerFlags = int(data[0])
	info.SkillLevel = int(data[1])

	data = data[2:]
	v := NewVarIntFrom(data)
	info.NumPlayers = v.Unpack()
	t.Logf("NumPlayers: %d", info.NumPlayers)
	info.MaxPlayers = v.Unpack()
	t.Logf("MaxPlayers: %d", info.MaxPlayers)
	info.NumClients = v.Unpack()
	t.Logf("NumClients: %d", info.NumClients)
	info.MaxClients = v.Unpack()
	t.Logf("MaxClients: %d", info.MaxClients)
	info.Players = make([]*PlayerInfo, 0, info.NumClients)

	data = v.Data() // return the not yet used remaining data

	for i := 0; i < info.NumClients; i++ {
		player := PlayerInfo{}

		slots := bytes.SplitN(v.Data(), []byte("\x00"), 3)

		player.Name = string(slots[0])
		player.Clan = string(slots[1])

		v = NewVarIntFrom(slots[2])
		player.Country = v.Unpack()
		player.Score = v.Unpack()
		player.Type = v.Unpack()

		info.Players = append(info.Players, &player)
	}

	t.Log(info.String())
}

func TestFullServerInfo(t *testing.T) {
	msList := getMasterServerList()

	serverAddresses := make([]net.UDPAddr, 0, 1024)
	for _, ms := range msList {
		lst, err := ms.GetServerList()
		if err != nil {
			continue
		}
		serverAddresses = append(serverAddresses, lst...)
	}

	t.Logf("Retrieved %d server addresses", len(serverAddresses))

	connections := make([]*net.UDPConn, 0, len(serverAddresses))

	for _, addr := range serverAddresses {
		conn, err := net.DialUDP("udp", nil, &addr)
		if err != nil || conn == nil {
			continue
		}
		defer conn.Close()
		connections = append(connections, conn)
	}

	gameservers := make([]*GameServer, 0, len(connections))
	for idx, conn := range connections {
		if idx > 70 {
			break
		}
		bc, err := NewBrowserConnetion(conn, 30*time.Second)
		if err != nil {
			t.Error(err)
			continue
		}
		gameservers = append(gameservers, &GameServer{&bc})
	}

	for _, gs := range gameservers {
		_, err := gs.ServerInfo()
		if err != nil {
			t.Error(err)
			continue
		}

	}
}
