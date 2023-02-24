package ams

import "github.com/algorand/go-algorand-sdk/client/v2/algod"

type Wallet struct {
	ac *algod.Client
}

type WalletOption func(*Wallet)

func WithWalletAlgod(ac *algod.Client) WalletOption {
	return func(w *Wallet) {
		w.ac = ac
	}
}

func MakeWallet(opts ...WalletOption) (*Wallet, error) {
	w := &Wallet{}

	for _, opt := range opts {
		opt(w)
	}

	return w, nil
}

func (w *Wallet) Run() error {
	return nil
}
