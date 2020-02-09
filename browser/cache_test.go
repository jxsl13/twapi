package browser

import (
	"testing"
	"time"
)

func TestExpiringServerInfo_Expired(t *testing.T) {
	info := ServerInfo{}
	esi := ExpiringServerInfo{
		info,
		time.Now().Add(20 * time.Millisecond),
	}

	if esi.Expired() {
		t.Fatal("expired, even tho it should not have expired.")
	}

	time.Sleep(25 * time.Millisecond)

	if !esi.Expired() {
		t.Fatal("didn't expire, even tho it should have expired.")
	}
}
