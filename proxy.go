package ams

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/encoding/msgpack"
	"github.com/algorand/go-algorand-sdk/types"
	"github.com/dragmz/wc"
	"github.com/pkg/errors"
)

type ProxySigner struct {
	pa   []PeerAddr
	ma   *crypto.MultisigAccount
	addr string

	pcb   ProxyPeerAddrCallback
	debug bool
}

type ProxySignerOption func(s *ProxySigner)

type ProxyPeerAddrCallback func() []PeerAddr

func WithProxySignerDebug(debug bool) ProxySignerOption {
	return func(s *ProxySigner) {
		s.debug = debug
	}
}

func WithProxySignerMultisig(ma *crypto.MultisigAccount) ProxySignerOption {
	return func(s *ProxySigner) {
		s.ma = ma
	}
}

func WithProxySignerPeersCallback(cb ProxyPeerAddrCallback) ProxySignerOption {
	return func(s *ProxySigner) {
		s.pcb = cb
	}
}

func MakeProxySigner(addr string, opts ...ProxySignerOption) (*ProxySigner, error) {
	s := &ProxySigner{
		addr: addr,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

type peerPartial struct {
	Partial [][]byte
	Address string
}

type PeerAddr struct {
	Peer    *wc.Client
	Address string
}

func (s *ProxySigner) Sign(req wc.AlgoSignRequest) (*wc.AlgoSignResponse, error) {
	psch := make(chan peerPartial)
	ctx, cancel := context.WithCancelCause(context.Background())

	pa := s.pcb()
	for i, pu := range pa {
		if s.debug {
			fmt.Println("Peer sign request:", pu)
		}
		go func(i int, p PeerAddr) {
			err := func() error {
				if s.debug {
					fmt.Println("Requesting sign - peer:", p)
				}

				for i := 0; i < len(req.Params[0]); i++ {
					req.Params[0][i].AuthAddr = p.Address
				}

				pstxs, err := p.Peer.SignTransactions(req)
				if err != nil {
					return errors.Wrapf(err, "failed to sign transactions - addr: %s, peer: %v", p.Address, p.Peer)
				}

				select {
				case psch <- peerPartial{
					Partial: pstxs,
					Address: p.Address,
				}:
				case <-ctx.Done():
				}

				return nil
			}()

			if err != nil {
				cancel(err)
			}
		}(i, pu)
	}

	var all []peerPartial

	var min int
	if s.ma != nil {
		min = int(s.ma.Threshold)
	}
	if min == 0 {
		min = 1
	}

	if s.debug {
		fmt.Println("Awaiting responses:", min)
	}

	for i := 0; i < min; i++ {
		select {
		case pp := <-psch:
			if s.debug {
				fmt.Println("Received response - address:", pp.Address, ", len:", len(pp.Partial))
			}
			all = append(all, pp)
		case <-ctx.Done():
			return nil, errors.Wrap(ctx.Err(), "Failed to sign transactions")
		}
	}

	cancel(nil)

	merge := func() func([][]byte) ([]byte, error) {
		if s.ma != nil && s.ma.Threshold > 1 {
			return func(txs [][]byte) ([]byte, error) {
				_, stx, err := crypto.MergeMultisigTransactions(txs...)
				return stx, err
			}
		} else {
			return func(txs [][]byte) ([]byte, error) {
				if len(txs) == 0 {
					return nil, nil
				}
				return txs[0], nil
			}
		}
	}()

	var count int

	for _, pp := range all {
		curr := len(pp.Partial)
		if count < curr {
			count = curr
		}
	}

	for _, pp := range all {
		if len(pp.Partial) != count {
			return nil, errors.Errorf("Received invalid number of partial transactions - got: %d, expected: %d, addr: %s", len(pp.Partial), count, pp.Address)
		}
	}

	resp := wc.AlgoSignResponse{}

	for i := 0; i < count; i++ {
		var partial [][]byte

		for _, pp := range all {
			partial = append(partial, pp.Partial[i])
		}

		if s.ma != nil {
			maddr, err := s.ma.Address()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get multisig address")
			}

			for i, bs := range partial {
				var txn types.SignedTxn
				err := msgpack.Decode(bs, &txn)
				if err != nil {
					return nil, errors.Wrap(err, "failed to decode signed transaction msgpack")
				}

				if txn.AuthAddr.IsZero() {
					// TODO: I'm not sure about it but needed for Exodus that does not set it
					txn.AuthAddr = maddr

					bs = msgpack.Encode(txn)
				}

				mstx, err := ConvertToMultisig(bs, *s.ma)
				if err != nil {
					return nil, errors.Wrap(err, "failed to convert to multisig")
				}
				partial[i] = mstx
			}
		}

		stx, err := merge(partial)
		if err != nil {
			return nil, errors.Wrap(err, "failed to merge partial transactions")
		}

		b64 := base64.StdEncoding.EncodeToString(stx)
		resp.Result = append(resp.Result, b64)
	}

	if s.debug {
		fmt.Println("Response:", resp)
	}

	return &resp, nil
}

func (s *ProxySigner) Address() string {
	return s.addr
}
