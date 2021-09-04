package browser

import (
	"bytes"
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

// Equal compares two instances for equality.
func (p *PlayerInfo) Equal(other PlayerInfo) bool {
	return p.Name == other.Name && p.Clan == other.Clan && p.Type == other.Type && p.Country == other.Country && p.Score == other.Score

}

func (p *PlayerInfo) String() string {
	b, _ := json.Marshal(p)
	return string(b)
}

// marshalBinary returns a binary representation of the PlayerInfo
// no delimiter is appended at the end of the byte slice
func (p *PlayerInfo) MarshalBinary() ([]byte, error) {

	data := make([]byte, 0, 2*len(delimiter)+len(p.Name)+len(p.Clan)+3*5)

	data = append(data, []byte(p.Name)...)
	data = append(data, delimiter...)

	data = append(data, []byte(p.Clan)...)
	data = append(data, delimiter...)

	var v compression.VarInt
	v.Pack(p.Country)
	v.Pack(p.Score)
	v.Pack(p.Type)

	data = append(data, v.Bytes()...)
	return data, nil
}

func (p *PlayerInfo) UnmarshalBinary(data []byte) error {
	slots := bytes.SplitN(data, delimiter, 3) // create 3 slots
	if len(slots) != 3 {
		return fmt.Errorf("%w : expected slots: 3 got: %d", ErrMalformedResponseData, len(slots))
	}

	p.Name = string(slots[0])
	p.Clan = string(slots[1])
	var err error
	v := compression.NewVarIntFrom(slots[2])
	p.Country, err = v.Unpack()
	if err != nil {
		return err
	}
	p.Score, err = v.Unpack()
	if err != nil {
		return err
	}
	p.Type, err = v.Unpack()
	if err != nil {
		return err
	}
	return nil
}

// PlayerInfos must be pre allocated in order for the unmarshaler to know the
// number of playerinfos
type PlayerInfos []PlayerInfo

func (pi PlayerInfos) UnmarshalBinary(data []byte) error {
	v := compression.NewVarIntFrom(data)
	var err error
	for idx, player := range pi {

		slots := bytes.SplitN(v.Bytes(), delimiter, 3) // create 3 slots
		if len(slots) != 3 {
			return fmt.Errorf("%w : expected slots: 3 got: %d", ErrMalformedResponseData, len(slots))
		}

		player.Name = string(slots[0])
		player.Clan = string(slots[1])

		v = compression.NewVarIntFrom(slots[2])
		player.Country, err = v.Unpack()
		if err != nil {
			return err
		}
		player.Score, err = v.Unpack()
		if err != nil {
			return err
		}
		player.Type, err = v.Unpack()
		if err != nil {
			return err
		}

		pi[idx] = player
	}
	return nil
}
