package ams

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseEmptyAddrs(t *testing.T) {
	addrs, err := ParseAddrs("", ",")
	assert.NoError(t, err)

	assert.Len(t, addrs, 0)
}

func TestParseAddr(t *testing.T) {
	addrs, err := ParseAddrs("QFCRYKFTUI3RYCO4SSKJTU6VOYCKIW2KPYMAB37VYG4WRCEGEMM2FDJ4YQ", ",")
	assert.NoError(t, err)

	assert.Len(t, addrs, 1)
}

func TestParseAddrs(t *testing.T) {
	addrs, err := ParseAddrs("QFCRYKFTUI3RYCO4SSKJTU6VOYCKIW2KPYMAB37VYG4WRCEGEMM2FDJ4YQ,4D2VPFW5IGRJZYQURHIR6DWKYUWUI3MYTJKAKMTPKQU5R3PROZASZBFOHQ", ",")
	assert.NoError(t, err)

	assert.Len(t, addrs, 2)
}
