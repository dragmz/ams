package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/encoding/msgpack"
	"github.com/algorand/go-algorand-sdk/types"
	"github.com/dragmz/ams"
	"github.com/dragmz/tqr"
	"github.com/dragmz/wc"
	"github.com/pkg/errors"
)

type args struct {
	Algod      string
	AlgodToken string

	Paths pathsArg

	Uri       string
	Address   string
	Threshold uint

	Debug bool
}

func run(a args) error {
	if len(a.Paths) == 0 && len(a.Uri) == 0 {
		return errors.New("no transaction source specified")
	}

	if a.Threshold == 0 {
		return errors.New("threshold must be >= 0")
	}

	accs, err := ams.ParseAddrs(a.Address, ",")
	if err != nil {
		return err
	}

	if len(accs) > 0 && len(accs) < int(a.Threshold) {
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

	var pa []ams.PeerAddr
	var tries uint

	for len(pa) < int(a.Threshold) {
		fmt.Printf("Signers - need: %d / %d, got: %d, tries: %d:\n", a.Threshold, len(accs), len(pa), tries)

		peer, res, err := wc.MakeClient(
			meta,
			wc.WithClientDebug(a.Debug),
			wc.WithClientUrlHandler(func(uri wc.Uri) error {
				uch <- uri
				return nil
			}))
		if err != nil {
			return errors.Wrap(err, "failed to make client")
		}

		tries++

		// TODO: supports first address only
		for _, addr := range res.Accounts {
			if addrs_left[addr] || len(accs) == 0 {
				delete(addrs_left, addr)
				pa = append(pa, ams.PeerAddr{
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

	s, err := ams.MakeProxySigner(addr,
		ams.WithProxySignerDebug(a.Debug),
		ams.WithProxySignerMultisig(ma),
		ams.WithProxySignerPeersCallback(func() []ams.PeerAddr {
			return pa
		}),
	)

	if err != nil {
		return errors.Wrap(err, "failed to create proxy signer")
	}

	if len(a.Paths) > 0 {
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

		fmt.Println("Press Enter to sign transactions..")
		rdr := bufio.NewReader(os.Stdin)
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

		fmt.Println("Sending signed transactions..")

		if a.Debug {
			fmt.Println(resp)
		}

		ac, err := algod.MakeClient(a.Algod, a.AlgodToken)
		if err != nil {
			return errors.Wrap(err, "failed to make algod client")
		}

		var group []byte

		for _, b64 := range resp.Result {
			bs, err := base64.StdEncoding.DecodeString(b64)
			if err != nil {
				return errors.Wrap(err, "failed to decode transaction")
			}

			group = append(group, bs...)
		}

		id, err := ac.SendRawTransaction(group).Do(context.Background())
		if err != nil {
			return errors.Wrap(err, "failed to send transactions")
		}

		fmt.Println("Id:", id)
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

	flag.StringVar(&a.Algod, "algod", "https://mainnet-api.algonode.cloud", "algod node address")
	flag.StringVar(&a.AlgodToken, "algod-token", "", "algod node token")
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
