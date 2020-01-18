package main

import (
	"net"
	"strings"
	"testing"
)

func TestMasterServers(t *testing.T) {
	ms := getMasterServerList()

	for _, s := range ms {
		host := strings.Split(s, ":")[0]
		ips, err := net.LookupIP(host)

		if err != nil {
			t.Errorf("%s %s", host, err.Error())
		} else {
			t.Log(host, ips[0])
		}
	}
}
