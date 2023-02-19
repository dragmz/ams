package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/encoding/msgpack"
	"github.com/algorand/go-algorand-sdk/mnemonic"
	"github.com/algorand/go-algorand-sdk/types"
	"github.com/dragmz/ams"
	"github.com/dragmz/wc"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ed25519"
)

type args struct {
	Uri       string
	Address   string
	Mnemonic  string
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

func signTransctionFromInput(txn types.Transaction, rdr *bufio.Reader) ([]byte, error) {
	fmt.Println("Transaction base32:")

	bs := msgpack.Encode(txn)

	fmt.Println(ams.TxnTransferEncoding.EncodeToString(bs))
	fmt.Println("Enter signed transaction base32 (or file path with < prefix, e.g. <txn1):")

	pstx32, err := ams.ReadInput(rdr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read transaction input")
	}

	pstx, err := ams.TxnTransferEncoding.DecodeString(pstx32)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode transaction")
	}

	return pstx, nil
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

	var sks []ed25519.PrivateKey

	if a.Mnemonic != "" {
		mnems := strings.Split(a.Mnemonic, ",")
		for _, mnem := range mnems {
			sk, err := mnemonic.ToPrivateKey(mnem)
			if err != nil {
				return err
			}

			sks = append(sks, sk)
		}
	}

	if a.Debug {
		fmt.Println("Sks:", len(sks))
	}

	u, err := wc.ParseUri(a.Uri)
	if err != nil {
		return err
	}

	if a.Debug {
		fmt.Println("Uri:", u)
	}

	c, err := wc.MakeConn(wc.WithKey(u.Key), wc.WithHost(u.Url.Host), wc.WithDebug(a.Debug))
	if err != nil {
		return err
	}

	err = c.Subscribe(u.Topic)
	if err != nil {
		return err
	}

	rdr := bufio.NewReader(os.Stdin)

	var dapp string
	for {
		r, err := c.Read()
		if err != nil {
			return err
		}

		if r.Method == "" {
			// response
			if a.Debug {
				fmt.Println("GOT RESPONSE")
			}
		} else {
			// request
			if a.Debug {
				fmt.Println("GOT REQUEST")
			}

			switch r.Method {
			case "algo_signTxn":
				var req AlgoSignTxnRequest
				err = json.Unmarshal(r.Result, &req)
				if err != nil {
					return err
				}

				skips := make([]bool, len(req.Params[0]))
				txs := make([]types.Transaction, len(req.Params[0]))

				for i, item := range req.Params[0] {
					bs, err := base64.StdEncoding.DecodeString(item.Txn)
					if err != nil {
						return err
					}

					var txn types.Transaction
					err = msgpack.Decode(bs, &txn)
					if err != nil {
						return err
					}

					if item.Signers != nil && len(item.Signers) == 0 {
						skips[i] = true
					}

					if txn.Sender.String() != addr {
						skips[i] = true
					}

					txs[i] = txn

					if a.Debug {
						fmt.Println("Txn:", i, ", skipped:", skips[i])
					}
					fmt.Println(ams.FormatTxn(txn))
				}

				fmt.Println("Press enter to sign transactions..")
				rdr.ReadString('\n')

				b64stxs := make([]string, len(txs))
				for i, txn := range txs {
					if skips[i] {
						continue
					}

					var stx []byte
					var pstxs [][]byte

					for i, sk := range sks {
						if a.Debug {
							fmt.Printf("Signing with sk #%d\n", i)
						}

						var pstx []byte

						if len(accs) > 1 {
							_, pstx, err = crypto.SignMultisigTransaction(sk, ma, txn)
							if err != nil {
								return errors.Wrap(err, "failed to sign multisig transaction")
							}
						} else {
							_, pstx, err = crypto.SignTransaction(sks[0], txn)
							if err != nil {
								return errors.Wrap(err, "failed to sign transaction")
							}
						}

						pstxs = append(pstxs, pstx)
					}

					for len(pstxs) < int(a.Threshold) {
						pstx, err := signTransctionFromInput(txn, rdr)
						if err != nil {
							return errors.Wrap(err, "failed to sign transaction from input")
						}

						pstxs = append(pstxs, pstx)
					}

					if len(accs) > 1 && a.Threshold > 1 {
						_, stx, err = crypto.MergeMultisigTransactions(pstxs...)
						if err != nil {
							return errors.Wrap(err, "failed to merge multisig transactions")
						}
					} else {
						stx = pstxs[0]
					}

					b64stx := base64.StdEncoding.EncodeToString(stx)
					b64stxs[i] = b64stx
				}

				response := wc.OutgoingResponse{
					Header: wc.MakeResponseHeader(r.Id),
					Result: b64stxs,
				}

				err = c.Send(dapp, response)

				if err != nil {
					return err
				}

			case "wc_sessionRequest":
				var req wc.SessionRequestRequest
				err = json.Unmarshal(r.Result, &req)
				if err != nil {
					return err
				}

				peer, err := wc.MakeTopic()
				if err != nil {
					return err
				}

				err = c.Subscribe(peer)
				if err != nil {
					return err
				}

				result := wc.SessionRequestResponseResult{
					PeerId: peer,
					PeerMeta: wc.SessionRequestPeerMeta{
						Description: "Algorand Multisig Tools",
						Url:         "https://github.com/dragmz/ams",
						Name:        "AMS"},
					Approved: true,
					ChainId:  4160,
					Accounts: []string{
						addr,
					},
				}

				response := wc.OutgoingResponse{
					Header: wc.MakeResponseHeader(r.Id),
					Result: result,
				}

				dapp = req.Params[0].PeerId

				err = c.Send(dapp, response)
				if err != nil {
					return err
				}
			}
		}
	}
}

func main() {
	var a args

	flag.StringVar(&a.Uri, "uri", "", "WalletConnect uri")
	flag.StringVar(&a.Address, "addr", "", "Algorand account address")
	flag.StringVar(&a.Mnemonic, "mnemonic", "", "Signer mnemonic")
	flag.UintVar(&a.Threshold, "threshold", 1, "Multisig threshold")
	flag.BoolVar(&a.Debug, "debug", false, "debug mode")
	flag.Parse()

	err := run(a)
	if err != nil {
		panic(err)
	}
}
