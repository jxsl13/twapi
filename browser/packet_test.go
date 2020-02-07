package browser

import (
	"reflect"
	"testing"
)

func TestNewServerListRequestPacket(t *testing.T) {
	type args struct {
		t Token
	}
	tests := []struct {
		name    string
		args    args
		want    ServerListRequestPacket
		wantErr bool
	}{
		{"expired token", args{Token{}}, ServerListRequestPacket{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewServerListRequestPacket(tt.args.t)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewServerListRequestPacket() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewServerListRequestPacket() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewServerCountRequestPacket(t *testing.T) {
	type args struct {
		t Token
	}
	tests := []struct {
		name    string
		args    args
		want    ServerCountRequestPacket
		wantErr bool
	}{
		{"expired token", args{Token{}}, ServerCountRequestPacket{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewServerCountRequestPacket(tt.args.t)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewServerCountRequestPacket() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewServerCountRequestPacket() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewServerInfoRequestPacket(t *testing.T) {
	type args struct {
		t Token
	}
	tests := []struct {
		name    string
		args    args
		want    ServerInfoRequestPacket
		wantErr bool
	}{
		{"expired token", args{Token{}}, ServerInfoRequestPacket{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewServerInfoRequestPacket(tt.args.t)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewServerInfoRequestPacket() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewServerInfoRequestPacket() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseToken(t *testing.T) {
	type args struct {
		serverResponse []byte
	}
	tests := []struct {
		name    string
		args    args
		want    Token
		wantErr bool
	}{
		{"too short token", args{[]byte{0, 1, 2, 3, 4, 5}}, Token{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseToken(tt.args.serverResponse)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseToken() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseServerInfo(t *testing.T) {
	type args struct {
		serverResponse []byte
		address        string
	}
	tests := []struct {
		name     string
		args     args
		wantInfo ServerInfo
		wantErr  bool
	}{
		{"input too short", args{[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}, "127.0.0.1:8303"}, ServerInfo{}, true},
		{"invalid input", args{[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18}, "abcd:8303"}, ServerInfo{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotInfo, err := ParseServerInfo(tt.args.serverResponse, tt.args.address)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseServerInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotInfo, tt.wantInfo) {
				t.Errorf("ParseServerInfo() = %v, want %v", gotInfo, tt.wantInfo)
			}
		})
	}
}
