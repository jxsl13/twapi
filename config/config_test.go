package config

import (
	"os"
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	b, err := os.ReadFile("./tests/autoexec.cfg")
	if err != nil {
		t.Fatal(err)
	}

	c := NewConfig()

	err = c.UnmarshalText(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(c) == 0 {
		t.Fatalf("parsed 0 commands")
	}

	//t.Errorf("commands parsed: %d/n%s", len(c), c.String())
}

func TestParseLine(t *testing.T) {

	tests := []struct {
		name    string
		data    string
		want    Command
		wantErr bool
	}{
		{"#1", `sv_name "some server #1"`, Command{Name: "sv_name", Args: []string{"some server #1"}}, false},
		{"#2", `  exec   "configs/shared-gctf.cfg" `, Command{Name: "exec", Args: []string{"configs/shared-gctf.cfg"}}, false},
		{"#3", `sv_port   8323  `, Command{Name: "sv_port", Args: []string{"8323"}}, false},
		{"#4", "   \t  # 12345", Command{}, true},
		{"#5", `add_vote "----- Flag Points -----" "say Flag Points"`,
			Command{
				Name: "add_vote",
				Args: []string{
					"----- Flag Points -----",
					"say Flag Points",
				},
			},
			false,
		},
		{"#6", `add_vote " 1" "sv_flag_points 1;peter 12"`, Command{
			Name: "add_vote",
			Args: []string{
				" 1",
				"sv_flag_points 1;peter 12",
			},
		},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := []byte(tt.data)
			var got Command
			err := got.UnmarshalText(data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseLine() = %v, want %v", got, tt.want)
			}
		})
	}
}
