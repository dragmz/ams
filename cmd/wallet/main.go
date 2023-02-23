package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/encoding/msgpack"
	"github.com/algorand/go-algorand-sdk/types"
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
	Paths     pathsArg
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
	pa   []peerAddr
	ma   *crypto.MultisigAccount
	addr string

	debug bool
}

type peerPartial struct {
	Partial [][]byte
	Address string
}

type peerAddr struct {
	Peer    *wc.Peer
	Address string
}

func (s *proxySigner) Sign(req wc.AlgoSignRequest) (*wc.AlgoSignResponse, error) {
	psch := make(chan peerPartial)
	ctx, cancel := context.WithCancelCause(context.Background())

	for i, pu := range s.pa {
		if s.debug {
			fmt.Println("Peer sign request:", pu)
		}
		go func(i int, p peerAddr) {
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
			return nil, errors.Errorf("Received invalid number of partial transactions - got: %d, expected: %d, addr:", len(pp.Partial), count, pp.Address)
		}
	}

	resp := wc.AlgoSignResponse{}

	for i := 0; i < count; i++ {
		var partial [][]byte

		for _, pp := range all {
			partial = append(partial, pp.Partial[i])
		}

		if s.ma != nil {
			for i, bs := range partial {
				var txn types.SignedTxn
				err := msgpack.Decode(bs, &txn)
				if err != nil {
					return nil, errors.Wrap(err, "failed to decode signed transaction msgpack")
				}

				if txn.AuthAddr.IsZero() {
					a, err := types.DecodeAddress(all[i].Address)
					if err != nil {
						return nil, errors.Wrap(err, "failed to decode addr")
					}

					// TODO: I'm not sure about it but needed for Exodus that does not set it
					txn.AuthAddr = a

					bs = msgpack.Encode(txn)
				}

				mstx, err := ams.ConvertToMultisig(bs, *s.ma)
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
		if a.Threshold != 1 {
			return errors.New("missing address")
		}
	} else if len(accs) < int(a.Threshold) {
		return errors.New("number of addresses is less than the threshold")
	}

	var addr string
	var ma *crypto.MultisigAccount

	switch len(accs) {
	case 0:
	case 1:
		addr = accs[0].String()
	default:
		mma, err := crypto.MultisigAccountWithParams(1, uint8(a.Threshold), accs)
		if err != nil {
			return err
		}
		ad, err := mma.Address()
		if err != nil {
			return err
		}

		addr = ad.String()
		ma = &mma

		fmt.Println("Multisig:", addr)
	}

	var u *wc.Uri
	if len(a.Uri) > 0 {
		uu, err := wc.ParseUri(a.Uri)
		if err != nil {
			return err
		}

		// TODO: refactor wc.ParseUri to return *
		u = &uu

		if a.Debug {
			fmt.Println("Uri:", u)
		}
	}

	meta := wc.SessionRequestPeerMeta{
		Name: "AMS",
	}

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

	addrs_left := map[string]bool{}
	for _, acc := range accs {
		addrs_left[acc.String()] = true
	}

	var pa []peerAddr
	var tries uint

	for len(pa) < int(a.Threshold) {
		fmt.Printf("Signers - need: %d / %d, got: %d, tries: %d:\n", a.Threshold, len(accs), len(pa), tries)

		client, err := wc.MakeClient(
			wc.WithClientDebug(a.Debug),
			wc.WithClientUrlHandler(func(uri wc.Uri) error {
				uch <- uri
				return nil
			}))
		if err != nil {
			return errors.Wrap(err, "failed to make client")
		}

		peer, res, err := client.RequestSession(meta)
		if err != nil {
			return errors.Wrap(err, "failed to request session")
		}

		tries++

		// TODO: supports first address only
		for _, addr := range res.Accounts {
			if addrs_left[addr] || len(accs) == 0 {
				delete(addrs_left, addr)
				pa = append(pa, peerAddr{
					Peer:    peer,
					Address: addr,
				})
				break
			}
		}

		// TODO: disconnect if not accepted
	}

	if len(addr) == 0 {
		addr = pa[0].Address
	}

	rdr := bufio.NewReader(os.Stdin)

	s := &proxySigner{
		r:    rdr,
		pa:   pa,
		ma:   ma,
		addr: addr,

		debug: a.Debug,
	}

	if u != nil {
		w, err := wc.MakeServer(*u, s,
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
	} else if len(a.Paths) > 0 {
		var txs []types.Transaction

		for _, p := range a.Paths {
			bs, err := os.ReadFile(p)
			if err != nil {
				return errors.Wrap(err, "failed to read transaction from file")
			}

			var ustx types.SignedTxn
			err = msgpack.Decode(bs, &ustx)
			if err != nil {
				return errors.Wrap(err, "failed to decode transaction msgpack")
			}

			// TODO: check if signed

			txs = append(txs, ustx.Txn)
		}

		for _, txn := range txs {
			fmt.Println(ams.FormatTxn(txn))
		}

		fmt.Println("Press Enter to sign the transactions..")
		rdr.ReadString('\n')

		req := wc.AlgoSignRequest{
			Params: [][]wc.AlgoSignParams{
				{},
			},
		}

		for _, txn := range txs {
			b64 := base64.StdEncoding.EncodeToString(msgpack.Encode(txn))
			req.Params[0] = append(req.Params[0], wc.AlgoSignParams{
				TxnBase64: b64,
			})
		}

		resp, err := s.Sign(req)
		if err != nil {
			return errors.Wrap(err, "failed to sign transactions")
		}

		fmt.Println("Signed transactions:")
		fmt.Println(resp)
	} else {
		return errors.New("No transactions input specified. Must be either uri or path(s).")
	}

	return nil
}

type pathsArg []string

func (i *pathsArg) String() string {
	return strings.Join(*i, ",")
}

func (i *pathsArg) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var myFlags pathsArg

func main() {
	var a args

	flag.StringVar(&a.Uri, "uri", "", "WalletConnect uri")
	flag.StringVar(&a.Address, "addr", "", "Algorand account address")
	flag.UintVar(&a.Threshold, "threshold", 1, "Multisig threshold")
	flag.BoolVar(&a.Debug, "debug", false, "debug mode")
	flag.Var(&a.Paths, "path", "transactions input paths")
	flag.Parse()

	err := run(a)
	if err != nil {
		panic(err)
	}
}
