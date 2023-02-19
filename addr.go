package ams

import (
	"strings"

	"github.com/algorand/go-algorand-sdk/types"
)

func ParseAddrs(addrs string, sep string) ([]types.Address, error) {
	var accs []types.Address

	if len(addrs) > 0 {
		parts := strings.Split(addrs, sep)

		for _, addr := range parts {
			acc, err := types.DecodeAddress(addr)
			if err != nil {
				return nil, err
			}
			accs = append(accs, acc)
		}
	}

	return accs, nil
}
