package ams

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"sync/atomic"

	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/encoding/msgpack"
	"github.com/algorand/go-algorand-sdk/types"
	"github.com/dragmz/wc"
	"github.com/pkg/errors"
)

type FsRunner struct {
	paths []string
	used  atomic.Bool

	s     wc.Signer
	debug bool

	ac *algod.Client
}

type FsRunnerOption func(r *FsRunner)

func WithFsRunnerAlgod(ac *algod.Client) FsRunnerOption {
	return func(r *FsRunner) {
		r.ac = ac
	}
}

func WithFsRunnerDebug(debug bool) FsRunnerOption {
	return func(r *FsRunner) {
		r.debug = debug
	}
}

func WithFsRunnerSigner(s wc.Signer) FsRunnerOption {
	return func(r *FsRunner) {
		r.s = s
	}
}

func MakeFsRunner(paths []string, opts ...FsRunnerOption) (*FsRunner, error) {
	r := &FsRunner{
		paths: paths,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r, nil
}

func ReadRequestFromFiles(paths []string) (*wc.AlgoSignRequest, error) {
	var txs []types.Transaction

	for _, p := range paths {
		bs, err := os.ReadFile(p)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read transaction from file")
		}

		var ustx types.SignedTxn
		err = msgpack.Decode(bs, &ustx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode transaction msgpack")
		}

		// TODO: check if signed

		txs = append(txs, ustx.Txn)
	}

	req := wc.AlgoSignRequest{
		Params: [][]wc.AlgoSignParams{
			{},
		},
	}

	for _, txn := range txs {
		fmt.Printf("%+v\n", txn)

		b64 := base64.StdEncoding.EncodeToString(msgpack.Encode(txn))
		req.Params[0] = append(req.Params[0], wc.AlgoSignParams{
			TxnBase64: b64,
		})
	}

	return &req, nil
}

func (r *FsRunner) Run() error {
	if r.used.Swap(true) {
		return nil
	}

	req, err := ReadRequestFromFiles(r.paths)
	if err != nil {
		return errors.Wrap(err, "failed to read transactions from files")
	}

	resp, err := r.s.Sign(*req)
	if err != nil {
		return errors.Wrap(err, "failed to sign transactions")
	}

	fmt.Println("Sending signed transactions..")

	if r.debug {
		fmt.Println(resp)
	}

	var group []byte

	for _, b64 := range resp.Result {
		bs, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return errors.Wrap(err, "failed to decode transaction")
		}

		group = append(group, bs...)
	}

	id, err := r.ac.SendRawTransaction(group).Do(context.Background())
	if err != nil {
		return errors.Wrap(err, "failed to send transactions")
	}

	fmt.Println("Id:", id)

	return nil
}
