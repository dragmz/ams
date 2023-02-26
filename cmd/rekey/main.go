package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/mnemonic"
	"github.com/algorand/go-algorand-sdk/transaction"
	"github.com/pkg/errors"
)

type args struct {
	Mnemonic string

	Algod      string
	AlgodToken string

	RekeyTo string
}

func run(a args) error {
	ac, err := algod.MakeClient(a.Algod, a.AlgodToken)
	if err != nil {
		return errors.Wrap(err, "failed to make algod client")
	}

	a.Mnemonic = strings.ReplaceAll(a.Mnemonic, ",", " ")

	sk, err := mnemonic.ToPrivateKey(a.Mnemonic)
	if err != nil {
		return errors.Wrap(err, "failed to convert mnemonic to private key")
	}

	acc, err := crypto.AccountFromPrivateKey(sk)
	if err != nil {
		return errors.Wrap(err, "failed to convert private key to account")
	}

	fmt.Println("Rekeyed address:", acc.Address)

	sp, err := ac.SuggestedParams().Do(context.Background())
	if err != nil {
		return errors.Wrap(err, "failed to get status")
	}

	tx, err := transaction.MakePaymentTxnWithFlatFee(acc.Address.String(), acc.Address.String(), transaction.MinTxnFee, 0, uint64(sp.FirstRoundValid), uint64(sp.LastRoundValid), nil, "", sp.GenesisID, sp.GenesisHash)
	if err != nil {
		return errors.Wrap(err, "failed to make tx")
	}

	err = tx.Rekey(a.RekeyTo)
	if err != nil {
		return errors.Wrap(err, "failed to set rekey to")
	}

	_, stx, err := crypto.SignTransaction(sk, tx)
	if err != nil {
		return errors.Wrap(err, "failed to sign tx")
	}

	txid, err := ac.SendRawTransaction(stx).Do(context.Background())
	if err != nil {
		return errors.Wrap(err, "failed to send tx")
	}

	fmt.Println("Transaction ID:", txid)

	return nil
}

func main() {
	var a args

	flag.StringVar(&a.Algod, "algod", "https://mainnet-api.algonode.cloud", "algod address")
	flag.StringVar(&a.AlgodToken, "algod-token", "", "algod token")
	flag.StringVar(&a.Mnemonic, "mnemonic", "", "mnemonic")
	flag.StringVar(&a.RekeyTo, "rekey-to", "", "rekey-to address")
	flag.Parse()

	err := run(a)
	if err != nil {
		panic(err)
	}
}
