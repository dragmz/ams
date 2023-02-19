package main

import (
	"bufio"
	"encoding/base32"
	"flag"
	"fmt"
	"os"
	"strings"

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

	var ma crypto.MultisigAccount
	if len(addrs) > 1 {
		ma, err = crypto.MultisigAccountWithParams(1, uint8(a.Threshold), addrs)
		if err != nil {
			return errors.Wrap(err, "failed to build multisig account")
		}

		addr, err := ma.Address()
		if err != nil {
			return errors.Wrap(err, "failed to get multisig address")
		}

		fmt.Println("Multisig address:", addr)
	} else {
		fmt.Println("Address:", addrs[0])
	}

	sk, err := mnemonic.ToPrivateKey(a.Mnemonic)
	if err != nil {
		return errors.Wrap(err, "failed to convert mnemonic to private key")
	}

	rdr := bufio.NewReader(os.Stdin)

	var txnstr = a.Txn
	if txnstr == "" {
		fmt.Println("Enter transaction base32:")
		txnstr, err = rdr.ReadString('\n')
		if err != nil {
			return errors.Wrap(err, "failed to read transaction data")
		}
		txnstr = strings.TrimSpace(txnstr)
	}

	bs, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(txnstr)
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

	var stx []byte

	if len(addrs) > 1 {
		_, stx, err = crypto.SignMultisigTransaction(sk, ma, tx)
	} else {
		_, stx, err = crypto.SignTransaction(sk, tx)
	}

	if err != nil {
		return errors.Wrap(err, "failed to sign transaction")
	}

	fmt.Println("Signed txn base32:")
	fmt.Println(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(stx))

	return nil
}

func main() {
	var a args

	flag.StringVar(&a.Mnemonic, "mnemonic", "", "private key mnemonic")
	flag.StringVar(&a.Addresses, "addr", "", "multisig addresses")
	flag.IntVar(&a.Threshold, "threshold", 0, "multisig threshold")
	flag.StringVar(&a.Txn, "txn", "", "base32 transaction data")

	flag.Parse()

	err := run(a)
	if err != nil {
		panic(err)
	}
}
