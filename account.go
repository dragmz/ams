package ams

import (
	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/mnemonic"
	"github.com/pkg/errors"
)

type AccountSource struct {
	m   string
	pkp string
}

type AccountSourceOption func(s *AccountSource)

func WithAccountSourceMnemonic(m string) AccountSourceOption {
	return func(s *AccountSource) {
		s.m = m
	}
}

func WithAccountSourcePrivateKeyPath(path string) AccountSourceOption {
	return func(s *AccountSource) {
		s.pkp = path
	}
}

func MakeAccountSource(opts ...AccountSourceOption) (*AccountSource, error) {
	s := &AccountSource{}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

func (s *AccountSource) ReadAccount() (*crypto.Account, error) {
	srcs := 0

	if len(s.m) > 0 {
		srcs++
	}

	if len(s.pkp) > 0 {
		srcs++
	}

	if len(s.m) > 0 {
		sk, err := mnemonic.ToPrivateKey(s.m)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert mnemonic to private key")
		}

		acc, err := crypto.AccountFromPrivateKey(sk)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert private key to account")
		}

		return &acc, nil
	}

	if len(s.pkp) > 0 {
		password, err := ReadPasswordFromStdin(false)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read password")
		}

		acc, err := ReadAccountFromFile(s.pkp, password)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read account from file")
		}

		return acc, nil
	}

	return nil, errors.New("no account source specified")
}
