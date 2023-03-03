package ams

import (
	"fmt"

	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/pkg/errors"
)

type AddressSource struct {
	value     string
	threshold int

	ma   *crypto.MultisigAccount
	addr string
}

type AddressSourceOption func(s *AddressSource)

func WithAddressString(value string) AddressSourceOption {
	return func(s *AddressSource) {
		s.value = value
	}
}

func WithAddressThreshold(threshold int) AddressSourceOption {
	return func(s *AddressSource) {
		s.threshold = threshold
	}
}

func (s *AddressSource) Address() string {
	return s.addr
}

func (s *AddressSource) Multisig() *crypto.MultisigAccount {
	return s.ma
}

func MakeAddressSource(opts ...AddressSourceOption) (*AddressSource, error) {
	s := &AddressSource{}

	for _, opt := range opts {
		opt(s)
	}

	addrs, err := ParseAddrs(s.value, ",")
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse addresses")
	}

	if len(addrs) == 0 {
		return nil, errors.New("missing address")
	}

	if len(addrs) > 1 {
		ma, err := crypto.MultisigAccountWithParams(1, uint8(s.threshold), addrs)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build multisig account")
		}

		maddr, err := ma.Address()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get multisig address")
		}

		s.ma = &ma

		fmt.Println("Multisig address:", maddr)
		s.addr = maddr.String()
	} else {
		fmt.Println("Address:", addrs[0])
		s.addr = addrs[0].String()
	}

	return s, nil
}
