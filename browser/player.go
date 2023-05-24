package browser

import (
	"encoding/json"
	"fmt"

	"github.com/jxsl13/twapi/compression"
)

// PlayerInfo contains a players externally visible information
type PlayerInfo struct {
	Name    string `json:"name"`
	Clan    string `json:"clan"`
	Type    int    `json:"type"`
	Country int    `json:"country"`
	Score   int    `json:"score"`
}

func (p *PlayerInfo) String() string {
	b, _ := json.Marshal(p)
	return string(b)
}

// marshalBinary returns a binary representation of the PlayerInfo
// no delimiter is appended at the end of the byte slice
func (p *PlayerInfo) MarshalBinary() ([]byte, error) {

	packer := compression.NewPacker(make([]byte, 0, 2+len(p.Name)+len(p.Clan)+3*5))

	packer.AddString(p.Name)
	packer.AddString(p.Clan)

	packer.AddInt(p.Country)
	packer.AddInt(p.Score)
	packer.AddInt(p.Type)

	return packer.Bytes(), nil
}

func (p *PlayerInfo) UnmarshalBinary(data []byte) error {
	u := compression.NewUnpacker(data)
	var err error
	p.Name, err = u.NextString()
	if err != nil {
		return fmt.Errorf("failed to unmarshal name: %w", err)
	}

	p.Clan, err = u.NextString()
	if err != nil {
		return fmt.Errorf("failed to unmarshal clan: %w", err)
	}

	p.Country, err = u.NextInt()
	if err != nil {
		return fmt.Errorf("failed to unmarshal country code: %w", err)
	}
	p.Score, err = u.NextInt()
	if err != nil {
		return fmt.Errorf("failed to unmarshal score: %w", err)
	}
	p.Type, err = u.NextInt()
	if err != nil {
		return fmt.Errorf("failed to unmarshal player type: %w", err)
	}
	return err
}

// PlayerInfos must be pre allocated in order for the unmarshaler to know the
// number of playerinfos
type PlayerInfos []PlayerInfo

func (pi PlayerInfos) UnmarshalBinary(data []byte) error {
	u := compression.NewUnpacker(data)
	var err error
	for idx, player := range pi {

		player.Name, err = u.NextString()
		if err != nil {
			return fmt.Errorf("failed to unmarshal name: %w", err)
		}
		player.Clan, err = u.NextString()
		if err != nil {
			return fmt.Errorf("failed to unmarshal clan: %w", err)
		}

		player.Country, err = u.NextInt()
		if err != nil {
			return err
		}
		player.Score, err = u.NextInt()
		if err != nil {
			return err
		}
		player.Type, err = u.NextInt()
		if err != nil {
			return err
		}

		pi[idx] = player
	}
	return err
}
