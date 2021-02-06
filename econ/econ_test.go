package econ

import (
	"fmt"
	"testing"
)

var (
	hasValidCredentials = false
	validAddress        = "127.0.0.1:9303"
	validPassword       = "12345"
)

func TestDialTo(t *testing.T) {
	if !hasValidCredentials {

		_, err := DialTo(validAddress, validPassword)
		if err == nil {
			t.Fatal("expected error, because credentials are not valid")
		}
		return
	}

	conn, err := DialTo(validAddress, validPassword)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	for i := 0; i < 30; i++ {
		line, err := conn.ReadLine()
		if err != nil {
			t.Error(err)
		}
		fmt.Printf("Line: %s\n", line)
	}
}
