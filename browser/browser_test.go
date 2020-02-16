package browser

import "testing"

func TestServerInfo_Equal(t *testing.T) {
	type fields struct {
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
		Players     []PlayerInfo
	}
	type args struct {
		other ServerInfo
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{"equal empty", fields{}, args{ServerInfo{}}, true},
		{"not equal", fields{Address: "127.0.0.1:8303", Version: "0.7.4", Name: "Simply zCatch", MaxClients: 64, Players: []PlayerInfo{{Name: "player"}}}, args{ServerInfo{}}, false},
		{"equal", fields{Address: "127.0.0.1:8303", Version: "0.7.4", Name: "Simply zCatch", MaxClients: 64, Players: []PlayerInfo{{Name: "player1"}, {Name: "player2", Clan: "clan2"}}}, args{ServerInfo{Address: "127.0.0.1:8303", Version: "0.7.4", Name: "Simply zCatch", MaxClients: 64, Players: []PlayerInfo{{Name: "player1"}, {Name: "player2", Clan: "clan2"}}}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ServerInfo{
				Address:     tt.fields.Address,
				Version:     tt.fields.Version,
				Name:        tt.fields.Name,
				Hostname:    tt.fields.Hostname,
				Map:         tt.fields.Map,
				GameType:    tt.fields.GameType,
				ServerFlags: tt.fields.ServerFlags,
				SkillLevel:  tt.fields.SkillLevel,
				NumPlayers:  tt.fields.NumPlayers,
				MaxPlayers:  tt.fields.MaxPlayers,
				NumClients:  tt.fields.NumClients,
				MaxClients:  tt.fields.MaxClients,
				Players:     tt.fields.Players,
			}
			if got := s.Equal(tt.args.other); got != tt.want {
				t.Errorf("ServerInfo.Equal() = %v, want %v", got, tt.want)
			}
		})
	}

	address := "127.0.0.1:8303"

	info := ServerInfo{
		Address:    address,
		Version:    "0.7.4",
		Name:       "Simply zCatch",
		MaxClients: 64,
		Players:    []PlayerInfo{{Name: "player1"}, {Name: "player2", Clan: "clan2"}},
	}

	info2 := ServerInfo{
		Address:    address,
		Version:    "0.7.4",
		Name:       "Simply zCatch",
		MaxClients: 64,
		Players:    []PlayerInfo{{Name: "player1"}, {Name: "player3", Clan: "clan2"}},
	}

	if info.Equal(info2) {
		t.Fatal("both info and info2 are equal, but should not be.")
	}

	serverData, _ := info.MarshalBinary()
	data := make([]byte, tokenPrefixSize+len(sendInfoRaw), tokenPrefixSize+len(sendInfoRaw)+len(serverData))

	// add prefix
	i := 0
	for j := tokenPrefixSize; j < tokenPrefixSize+len(sendInfoRaw); j++ {
		data[j] = sendInfoRaw[i]
		i++
	}
	// add encoded server info with players
	data = append(data, serverData...)

	// parse the constructed data back into a server info
	parsedInfo, err := ParseServerInfo(data, address)

	// parsing failed
	if err != nil {
		t.Fatal(err)
	}

	// expected to be equal
	if !parsedInfo.Equal(info) {
		t.Fatalf("Wanted= %s, Parsed=%s", info.String(), parsedInfo.String())
	}
}
