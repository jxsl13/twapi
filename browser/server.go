package browser

import (
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
	ServerFlags byte        `json:"server_flags"`
	SkillLevel  byte        `json:"skill_level"`
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

// Equal compares two instances of ServerInfo and returns true if they are equal
func (s *ServerInfo) Equal(other ServerInfo) bool {

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
		if p != other.Players[idx] {
			return false
		}
	}
	return equalData
}

func (s *ServerInfo) String() string {
	b, _ := json.Marshal(s)
	return string(b)
}

// MarshalBinary returns a binary representation of the ServerInfo
func (s *ServerInfo) MarshalBinary() ([]byte, error) {

	p := compression.NewPacker()
	p.AddString(s.Version)
	p.AddString(s.Name)

	p.AddString(s.Hostname)
	p.AddString(s.Map)
	p.AddString(s.GameType)
	p.AddByte(s.ServerFlags)
	p.AddByte(s.SkillLevel)

	p.AddInt(s.NumPlayers)
	p.AddInt(s.MaxPlayers)
	p.AddInt(len(s.Players))
	p.AddInt(s.MaxClients)

	for _, player := range s.Players {
		playerData, _ := player.MarshalBinary()
		p.AddBytes(playerData)
	}

	return p.Bytes(), nil
}

// UnmarshalBinary creates a serverinfo from binary data
func (s *ServerInfo) UnmarshalBinary(data []byte) error {
	u := compression.NewUnpacker(data)

	var err error
	s.Version, err = u.NextString()
	if err != nil {
		return fmt.Errorf("failed to unmarshal version: %w", err)
	}

	s.Name, err = u.NextString()
	if err != nil {
		return fmt.Errorf("failed to unmarshal name: %w", err)
	}

	s.Hostname, err = u.NextString()
	if err != nil {
		return fmt.Errorf("failed to unmarshal hostname: %w", err)
	}

	s.Map, err = u.NextString()
	if err != nil {
		return fmt.Errorf("failed to unmarshal map: %w", err)
	}

	s.GameType, err = u.NextString()
	if err != nil {
		return fmt.Errorf("failed to unmarshal gametype: %w", err)
	}

	s.ServerFlags, err = u.NextByte()
	if err != nil {
		return fmt.Errorf("failed to unmarshal server flags: %w", err)
	}
	s.SkillLevel, err = u.NextByte()
	if err != nil {
		return fmt.Errorf("failed to unmarshal skill levels: %w", err)
	}

	s.NumPlayers, err = u.NextInt()
	if err != nil {
		return fmt.Errorf("failed to unmarshal number of players: %w", err)
	}
	s.MaxPlayers, err = u.NextInt()
	if err != nil {
		return fmt.Errorf("failed to unmarshal max players: %w", err)
	}
	s.NumClients, err = u.NextInt()
	if err != nil {
		return fmt.Errorf("failed to unmarshal number of clients: %w", err)
	}
	s.MaxClients, err = u.NextInt()
	if err != nil {
		return fmt.Errorf("failed to unmarshal max clients: %w", err)
	}

	// preallocation is needed for the unmarshaling to work
	s.Players = make(PlayerInfos, s.NumClients)
	return s.Players.UnmarshalBinary(u.Bytes())
}
