package browser

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/jxsl13/twapi/compression"
)

// ServerInfo contains the server's general information
type ServerInfo struct {
	Address     string      `json:"address"`
	Version     string      `json:"version"`
	Name        string      `json:"name"`
	Hostname    string      `json:"hostname,omitempty"`
	Map         string      `json:"map"`
	GameType    string      `json:"gametype"`
	ServerFlags int         `json:"server_flags"`
	SkillLevel  int         `json:"skill_level"`
	NumPlayers  int         `json:"num_players"`
	MaxPlayers  int         `json:"max_players"`
	NumClients  int         `json:"num_clients"`
	MaxClients  int         `json:"max_clients"`
	Players     PlayerInfos `json:"players"`
}

// Empty returns true if the whole struct does not contain any data at all
func (s *ServerInfo) Empty() bool {
	return s.Address == "" &&
		s.Version == "" &&
		s.Name == "" &&
		s.Hostname == "" &&
		s.Map == "" &&
		s.GameType == "" &&
		s.ServerFlags == 0 &&
		s.SkillLevel == 0 &&
		s.NumPlayers == 0 &&
		s.MaxPlayers == 0 &&
		s.NumClients == 0 &&
		s.MaxClients == 0 &&
		len(s.Players) == 0
}

// fix synchronizes the length of playerInfo with its struct field
func (s *ServerInfo) fix() {
	s.NumClients = len(s.Players)
}

// Equal compares two instances of ServerInfo and returns true if they are equal
func (s *ServerInfo) Equal(other ServerInfo) bool {
	s.fix()
	other.fix()
	equalData := s.Address == other.Address &&
		s.Version == other.Version &&
		s.Name == other.Name &&
		s.Hostname == other.Hostname &&
		s.Map == other.Map &&
		s.GameType == other.GameType &&
		s.ServerFlags == other.ServerFlags &&
		s.SkillLevel == other.SkillLevel &&
		s.NumPlayers == other.NumPlayers &&
		s.MaxPlayers == other.MaxPlayers &&
		s.NumClients == other.NumClients &&
		s.MaxClients == other.MaxClients

	// equal Players
	if len(s.Players) != len(other.Players) {
		return false
	}
	for idx, p := range s.Players {
		if !p.Equal(other.Players[idx]) {
			return false
		}
	}
	return equalData
}

func (s *ServerInfo) String() string {
	s.fix()
	b, _ := json.Marshal(s)
	return string(b)
}

// MarshalBinary returns a binary representation of the ServerInfo
func (s *ServerInfo) MarshalBinary() (data []byte, err error) {
	s.fix()
	// stack allocated
	d := [1500]byte{}
	data = d[:0]

	// pack
	data = append(data, []byte(s.Version)...)
	data = append(data, delimiter...)

	data = append(data, []byte(s.Name)...)
	data = append(data, delimiter...)

	data = append(data, []byte(s.Hostname)...)
	data = append(data, delimiter...)

	data = append(data, []byte(s.Map)...)
	data = append(data, delimiter...)

	data = append(data, []byte(s.GameType)...)
	data = append(data, delimiter...)

	data = append(data, byte(s.ServerFlags), byte(s.SkillLevel))

	var v compression.VarInt

	v.Pack(s.NumPlayers)
	v.Pack(s.MaxPlayers)
	v.Pack(len(s.Players)) // s.NumClients
	v.Pack(s.MaxClients)

	data = append(data, v.Bytes()...)
	v.Clear()

	for _, player := range s.Players {
		playerData, _ := player.MarshalBinary()
		data = append(data, playerData...)
	}

	return
}

// UnmarshalBinary creates a serverinfo from binary data
func (s *ServerInfo) UnmarshalBinary(data []byte) error {
	slots := bytes.SplitN(data, delimiter, 6) // create 6 slots
	if len(slots) != 6 {
		return fmt.Errorf("%w : expected slots: 6 got: %d", ErrMalformedResponseData, len(slots))
	}

	s.Version = string(slots[0])
	s.Name = string(slots[1])
	s.Hostname = string(slots[2])
	s.Map = string(slots[3])
	s.GameType = string(slots[4])

	data = slots[5] // get next raw data chunk

	s.ServerFlags = int(data[0])
	s.SkillLevel = int(data[1])

	data = data[2:] // skip first two already evaluated bytes

	v := compression.NewVarIntFrom(data)

	var err error
	s.NumPlayers, err = v.Unpack()
	if err != nil {
		return err
	}
	s.MaxPlayers, err = v.Unpack()
	if err != nil {
		return err
	}
	s.NumClients, err = v.Unpack()
	if err != nil {
		return err
	}
	s.MaxClients, err = v.Unpack()
	if err != nil {
		return err
	}

	// preallocation is needed for the unmarshaling to work
	s.Players = make(PlayerInfos, s.NumClients)
	return s.Players.UnmarshalBinary(v.Bytes()) //v.Bytes() returns the not yet used remaining data
}
