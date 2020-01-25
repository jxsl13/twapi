package main

import (
	"bytes"
	"fmt"
	"strings"
)

const (
	getInfo     = "\xff\xff\xff\xffgie3\x00" // need explicitly the trailing \x00
	receiveInfo = "\xff\xff\xff\xffinf3\x00"
)

// PlayerInfo contains a players externally visible information
type PlayerInfo struct {
	Name    string
	Clan    string
	Type    int
	Country int
	Score   int
}

func (p *PlayerInfo) String() string {
	return fmt.Sprintf("Name=%27s Clan=%13s Type=%1d Country=%3d Score=%6d\n", p.Name, p.Clan, p.Type, p.Country, p.Score)
}

// ServerInfo contains the server's general information
type ServerInfo struct {
	Address     string
	Version     string
	Name        string
	Hostname    string
	Map         string
	GameType    string
	ServerFlags int
	SkillLevel  int
	NumPlayers  int
	MaxPlayers  int
	NumClients  int
	MaxClients  int
	Players     []*PlayerInfo
}

func (s *ServerInfo) String() string {
	base := fmt.Sprintf("\nHostname: %10s\nAddress: %20s\nVersion: '%10s'\nName: '%s'\nGameType: %s\nMap: %s\nServerFlags: %b\nSkilllevel: %d\n%d/%d Players \n%d/%d Clients\n",
		s.Hostname,
		s.Address,
		s.Version,
		s.Name,
		s.GameType,
		s.Map,
		s.ServerFlags,
		s.SkillLevel,
		s.NumPlayers,
		s.MaxPlayers,
		s.NumClients,
		s.MaxClients)
	sb := strings.Builder{}
	sb.Grow(256 + s.NumClients*128)
	sb.WriteString(base)

	for _, p := range s.Players {
		sb.WriteString(p.String())
	}
	return sb.String()
}

// GameServer represents an ingame server that can fetch
// data and player infos
type GameServer struct {
	*BrowserConnection
}

// ServerInfo retrieves the server info from the underlying server
func (gs *GameServer) ServerInfo() (info ServerInfo, err error) {

	resp, err := gs.Request(getInfo, receiveInfo, 2048)
	if err != nil {
		return
	}
	if len(resp) == 0 {
		err = fmt.Errorf("ServerInfo(): reveived empty response")
		return
	}
	slots := bytes.SplitN(resp, []byte("\x00"), 6) // create 6 slots

	info.Address = gs.RemoteAddr().String()
	info.Version = string(slots[0])
	info.Name = string(slots[1])
	info.Hostname = string(slots[2])
	info.Map = string(slots[3])
	info.GameType = string(slots[4])

	data := slots[5] // get next raw data chunk

	info.ServerFlags = int(data[0])
	info.SkillLevel = int(data[1])

	data = data[2:] // skip first two already evaluated bytes
	v := NewVarIntFrom(data)
	info.NumPlayers = v.Unpack()
	info.MaxPlayers = v.Unpack()
	info.NumClients = v.Unpack()
	info.MaxClients = v.Unpack()

	// preallocate space for player pointers
	info.Players = make([]*PlayerInfo, 0, info.NumClients)

	data = v.Data() // return the not yet used remaining data

	for i := 0; i < info.NumClients; i++ {
		player := PlayerInfo{}

		slots := bytes.SplitN(v.Data(), []byte("\x00"), 3) // create 3 slots

		player.Name = string(slots[0])
		player.Clan = string(slots[1])

		v = NewVarIntFrom(slots[2])
		player.Country = v.Unpack()
		player.Score = v.Unpack()
		player.Type = v.Unpack()

		info.Players = append(info.Players, &player)
	}

	return
}
