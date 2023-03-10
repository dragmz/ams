package ams

import (
	"encoding/base64"

	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/encoding/msgpack"
	"github.com/algorand/go-algorand-sdk/types"
	"github.com/dragmz/wc"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ed25519"
)

type Signer interface {
	Sign(wc.AlgoSignRequest) (*wc.AlgoSignResponse, error)
	Address() string
}

type LocalSigner struct {
	sk    ed25519.PrivateKey
	ma    *crypto.MultisigAccount
	addr  string
	match string
}

type LocalSignerOption func(s *LocalSigner) error

func MakeLocalSigner(addr string, sk ed25519.PrivateKey, opts ...LocalSignerOption) (Signer, error) {
	s := &LocalSigner{}

	for _, opt := range opts {
		opt(s)
	}

	s.sk = sk
	s.addr = addr

	return s, nil
}

func WithLocalSignerMatchSender(match string) LocalSignerOption {
	return func(s *LocalSigner) error {
		s.match = match
		return nil
	}
}

func WithLocalSignerMultisigAccount(ma *crypto.MultisigAccount) LocalSignerOption {
	return func(s *LocalSigner) error {
		if ma != nil {
			maddr, err := ma.Address()
			if err != nil {
				return errors.Wrap(err, "failed to get multisig address")
			}

			s.addr = maddr.String()
		}

		s.ma = ma

		return nil
	}
}

func (s *LocalSigner) Address() string {
	return s.addr
}

func (s *LocalSigner) Sign(req wc.AlgoSignRequest) (*wc.AlgoSignResponse, error) {
	p := req.Params[0]
	var txs = make([]types.Transaction, len(p))

	for i, item := range p {
		bs, err := base64.StdEncoding.DecodeString(item.TxnBase64)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode base64 transactions data")
		}

		var txn types.Transaction
		err = msgpack.Decode(bs, &txn)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode transaction msgpack")
		}

		txs[i] = txn
	}

	var sign func(types.Transaction) (string, []byte, error)

	if s.ma != nil {
		sign = func(txn types.Transaction) (string, []byte, error) {
			return crypto.SignMultisigTransaction(s.sk, *s.ma, txn)
		}
	} else {
		sign = func(txn types.Transaction) (string, []byte, error) {
			return crypto.SignTransaction(s.sk, txn)
		}
	}

	res := make([][]byte, len(txs))

	for i, txn := range txs {
		if len(s.match) > 0 {
			if txn.Sender.String() != s.match {
				continue
			}
		}

		_, stx, err := sign(txn)
		if err != nil {
			return nil, errors.Wrap(err, "failed to sign transaction")
		}

		res[i] = stx
	}

	b64s := make([]string, len(res))
	for i, stx := range res {
		if stx == nil {
			continue
		}

		b64 := base64.StdEncoding.EncodeToString(stx)
		b64s[i] = b64
	}

	response := wc.AlgoSignResponse{
		Result: b64s,
	}

	return &response, nil
}
