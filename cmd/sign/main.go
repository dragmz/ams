package main

import (
	"bufio"
	"encoding/base64"
	"flag"
	"fmt"
	"os"

	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/encoding/msgpack"
	"github.com/algorand/go-algorand-sdk/mnemonic"
	"github.com/algorand/go-algorand-sdk/types"
	"github.com/dragmz/ams"
	"github.com/pkg/errors"
)

type args struct {
	Mnemonic  string
	Addresses string
	Threshold int
	Txn       string
}

func run(a args) error {
	addrs, err := ams.ParseAddrs(a.Addresses, ",")
	if err != nil {
		return errors.Wrap(err, "failed to parse addresses")
	}

	if len(addrs) == 0 {
		return errors.New("missing address")
	}

	ma, err := crypto.MultisigAccountWithParams(1, uint8(a.Threshold), addrs)
	if err != nil {
		return errors.Wrap(err, "failed to build multisig account")
	}

	addr, err := ma.Address()
	if err != nil {
		return errors.Wrap(err, "failed to get multisig address")
	}

	fmt.Println("Multisig address:", addr)

	sk, err := mnemonic.ToPrivateKey(a.Mnemonic)
	if err != nil {
		return errors.Wrap(err, "failed to convert mnemonic to private key")
	}

	rdr := bufio.NewReader(os.Stdin)

	var txnstr = a.Txn
	if txnstr == "" {
		fmt.Println("Enter transaction base64:")
		txnstr, err = rdr.ReadString('\n')
		if err != nil {
			return errors.Wrap(err, "failed to read transaction data")
		}
	}

	bs, err := base64.StdEncoding.DecodeString(txnstr)
	if err != nil {
		return errors.Wrap(err, "failed to decode transaction data")
	}

	var tx types.Transaction
	err = msgpack.Decode(bs, &tx)
	if err != nil {
		return errors.Wrap(err, "failed to decode transaction msgpack")
	}

	fmt.Println("- Transaction Details -")
	fmt.Println(ams.FormatTxn(tx))
	fmt.Println("Press Enter to sign the transaction..")
	rdr.ReadString('\n')

	_, stx, err := crypto.SignMultisigTransaction(sk, ma, tx)
	if err != nil {
		return errors.Wrap(err, "failed to sign transaction")
	}

	fmt.Println("Signed txn base64:")
	fmt.Println(base64.StdEncoding.EncodeToString(stx))

	return nil
}

func main() {
	var a args

	flag.StringVar(&a.Mnemonic, "mnemonic", "", "private key mnemonic")
	flag.StringVar(&a.Addresses, "addr", "", "multisig addresses")
	flag.IntVar(&a.Threshold, "threshold", 0, "multisig threshold")
	flag.StringVar(&a.Txn, "txn", "", "base64 transaction data")

	flag.Parse()

	err := run(a)
	if err != nil {
		panic(err)
	}
}
