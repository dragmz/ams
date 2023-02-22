package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"os"

	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/dragmz/ams"
	"github.com/dragmz/tqr"
	"github.com/dragmz/wc"
	"github.com/pkg/errors"
)

type args struct {
	Uri       string
	Address   string
	Threshold uint
	Debug     bool
}

type AlgoSignTxnRequestParams struct {
	Txn     string `json:"txn"`
	Signers []any  `json:"signers,omitempty"`
}

type AlgoSignTxnRequest struct {
	Params [][]AlgoSignTxnRequestParams `json:"params"`
}

type proxySigner struct {
	r    *bufio.Reader
	pus  []*wc.Peer
	ma   *crypto.MultisigAccount
	addr string

	debug bool
}

func (s *proxySigner) Sign(req wc.AlgoSignRequest) (*wc.AlgoSignResponse, error) {
	psch := make(chan [][]byte)
	ctx, cancel := context.WithCancelCause(context.Background())

	for _, pu := range s.pus {
		if s.debug {
			fmt.Println("Peer sign request:", pu)
		}
		go func(p *wc.Peer) {
			func() {
				pstxs, err := p.SignTransactions(req)
				if err != nil {
					cancel(errors.Wrap(err, "failed to sign transactions"))
				}

				select {
				case psch <- pstxs:
				case <-ctx.Done():
				}
			}()
		}(pu)
	}

	var all [][][]byte

	min := int(s.ma.Threshold)
	if min == 0 {
		min = 1
	}

	if s.debug {
		fmt.Println("Minimum peers:", min)
	}

	for i := 0; i < min; i++ {
		select {
		case partial := <-psch:
			if s.debug {
				fmt.Println("Received partials:", len(partial))
			}
			all = append(all, partial)
		case <-ctx.Done():
			return nil, errors.Wrap(ctx.Err(), "failed to sign")
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

	for _, partial := range all {
		curr := len(partial)
		if count < curr {
			count = curr
		}
	}

	for _, partial := range all {
		if len(partial) != count {
			return nil, errors.Errorf("Received invalid number of partial transactions - got: %d, expected: %d", len(partial), count)
		}
	}

	resp := wc.AlgoSignResponse{}

	for i := 0; i < count; i++ {
		var partial [][]byte

		for _, signer := range all {
			partial = append(partial, signer[i])
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

func (s *proxySigner) Address() string {
	return s.addr
}

func run(a args) error {
	if a.Threshold == 0 {
		return errors.New("threshold must be >= 0")
	}

	accs, err := ams.ParseAddrs(a.Address, ",")
	if err != nil {
		return err
	}

	if len(accs) == 0 {
		return errors.New("missing address")
	}

	if len(accs) < int(a.Threshold) {
		return errors.New("number of addresses is less than the threshold")
	}

	var addr string
	var ma crypto.MultisigAccount

	if len(accs) > 1 {
		ma, err = crypto.MultisigAccountWithParams(1, uint8(a.Threshold), accs)
		if err != nil {
			return err
		}
		ad, err := ma.Address()
		if err != nil {
			return err
		}

		addr = ad.String()

		fmt.Println("Multisig:", addr)
	} else {
		addr = accs[0].String()
	}

	u, err := wc.ParseUri(a.Uri)
	if err != nil {
		return err
	}

	if a.Debug {
		fmt.Println("Uri:", u)
	}

	meta := wc.SessionRequestPeerMeta{
		Name: "AMS",
	}

	ech := make(chan error)
	puch := make(chan *wc.Peer)

	min := len(ma.Pks)
	if min == 0 {
		min = 1
	}

	fmt.Printf("Signers sessions (need %d of %d):\n", min, len(accs))

	uch := make(chan wc.Uri)

	go func() {
		for {
			u, ok := <-uch
			if !ok {
				return
			}

			s := u.String()

			fmt.Println(tqr.New(s))
			fmt.Println(s)
		}
	}()

	for i := 0; i < min; i++ {
		go func(i int) {
			err := func() error {
				client, err := wc.MakeClient(
					wc.WithClientDebug(a.Debug),
					wc.WithClientUrlHandler(func(uri wc.Uri) error {
						uch <- uri
						return nil
					}))
				if err != nil {
					return errors.Wrap(err, "failed to make client")
				}

				peer, _, err := client.RequestSession(meta)
				if err != nil {
					return errors.Wrap(err, "failed to request session")
				}

				puch <- peer

				return nil
			}()

			if err != nil {
				ech <- err
			}
		}(i)
	}

	var pus []*wc.Peer

	for i := 0; i < min; i++ {
		select {
		case pu := <-puch:
			if a.Debug {
				fmt.Println("Peer:", pu)
			}
			pus = append(pus, pu)
		case err := <-ech:
			return errors.Wrap(err, "failed to request session")
		}
	}

	rdr := bufio.NewReader(os.Stdin)

	s := &proxySigner{
		r:    rdr,
		pus:  pus,
		ma:   &ma,
		addr: addr,

		debug: a.Debug,
	}

	w, err := wc.MakeServer(u, s,
		wc.WithServerDebug(a.Debug),
	)
	if err != nil {
		return errors.Wrap(err, "failed to make server")
	}

	for {
		err := w.Run()
		if err != nil {
			return err
		}
	}
}

func main() {
	var a args

	flag.StringVar(&a.Uri, "uri", "", "WalletConnect uri")
	flag.StringVar(&a.Address, "addr", "", "Algorand account address")
	flag.UintVar(&a.Threshold, "threshold", 1, "Multisig threshold")
	flag.BoolVar(&a.Debug, "debug", false, "debug mode")
	flag.Parse()

	err := run(a)
	if err != nil {
		panic(err)
	}
}
