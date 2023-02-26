package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"

	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/encoding/msgpack"
	"github.com/algorand/go-algorand-sdk/transaction"
	"github.com/algorand/go-algorand-sdk/types"
	"github.com/dragmz/ams"
	"github.com/dragmz/wc"
	"github.com/pkg/errors"
)

type args struct {
	Algod      string
	AlgodToken string

	Threshold uint

	Debug bool
}

func run(a args) error {
	ac, err := algod.MakeClient(a.Algod, a.AlgodToken)
	if err != nil {
		return errors.Wrap(err, "failed to create algod client")
	}

	dapp, err := wc.MakeConn(wc.WithConnDebug(a.Debug))
	if err != nil {
		return errors.Wrap(err, "failed to make wc conn")
	}

	acc := crypto.GenerateAccount()

	ma, err := crypto.MultisigAccountWithParams(1, uint8(a.Threshold), []types.Address{acc.Address})
	if err != nil {
		return errors.Wrap(err, "failed to make multisig account")
	}

	peer, res, err := wc.MakeClient(
		wc.SessionRequestPeerMeta{
			Name:        "wc",
			Description: "WalletConnect example 1",
		},
		wc.WithClientConn(dapp),
		wc.WithClientUrlHandler(func(uri wc.Uri) error {
			signer, err := ams.MakeLocalSigner(acc.Address.String(), acc.PrivateKey,
				ams.WithLocalSignerMultisigAccount(ma),
			)
			if err != nil {
				return errors.Wrap(err, "failed to make signer")
			}

			wallet, err := wc.MakeServer(uri, signer,
				wc.WithServerDebug(a.Debug),
			)
			if err != nil {
				return errors.Wrap(err, "failed to make wallet connection")
			}

			go func() error {
				defer wallet.Close()

				err := wallet.Run()
				if err != nil {
					return errors.Wrap(err, "failed to run wallet")
				}
				return nil
			}()

			return nil
		}))

	if err != nil {
		return errors.Wrap(err, "failed to make wc engine")
	}

	fmt.Println("DApp connecting to wallet..")

	if len(res.Accounts) == 0 {
		return errors.New("no accounts")
	}

	fmt.Println("Wallet addresses:", res.Accounts)

	account := res.Accounts[0]

	sp, err := ac.SuggestedParams().Do(context.Background())
	if err != nil {
		return errors.Wrap(err, "failed to get suggested params")
	}

	var txs []types.Transaction

	maddr, err := ma.Address()
	if err != nil {
		return errors.Wrap(err, "failed to get multisg address")
	}

	addr := maddr.String()

	for i := 0; i < 10; i++ {
		txn, err := transaction.MakePaymentTxnWithFlatFee(addr, account,
			transaction.MinTxnFee, 0, uint64(sp.FirstRoundValid), uint64(sp.LastRoundValid), []byte(fmt.Sprintf("test transaction %d", i)), "", sp.GenesisID, sp.GenesisHash)
		if err != nil {
			return errors.Wrap(err, "failed to make payment tx1")
		}

		txs = append(txs, txn)
	}

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

	resp, err := peer.Sign(context.Background(), req)
	if err != nil {
		return errors.Wrap(err, "failed to send txs")
	}

	raws, err := wc.DecodeAlgoSignResponse(*resp)
	if err != nil {
		return errors.Wrap(err, "failed to decode sign response")
	}

	fmt.Println("Signed transactions:", len(raws))

	return nil
}

func main() {
	var a args

	flag.StringVar(&a.Algod, "algod", "https://mainnet-api.algonode.cloud", "algod address")
	flag.StringVar(&a.AlgodToken, "algod-token", "", "algod token")
	flag.UintVar(&a.Threshold, "threshold", 1, "multisig threshold")
	flag.BoolVar(&a.Debug, "debug", false, "enable debug")

	flag.Parse()

	err := run(a)
	if err != nil {
		panic(err)
	}
}
