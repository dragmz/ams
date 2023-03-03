package main

import (
	"bufio"
	"encoding/base64"
	"flag"
	"fmt"
	"os"

	"github.com/algorand/go-algorand-sdk/encoding/msgpack"
	"github.com/algorand/go-algorand-sdk/types"
	"github.com/dragmz/ams"
	"github.com/dragmz/wc"
	"github.com/pkg/errors"
)

type args struct {
	Mnemonic    string
	Addr        string
	Threshold   int
	Txn         string
	Debug       bool
	Uri         string
	AuthAddr    string
	MatchSender string

	ClipboardUri bool

	PrivateKeyPath string
}

type manualConfirmSignerWrapper struct {
	s wc.Signer
	r *bufio.Reader
}

func (s *manualConfirmSignerWrapper) Sign(req wc.AlgoSignRequest) (*wc.AlgoSignResponse, error) {
	fmt.Println("Incoming transactions:")

	if len(req.Params) > 0 {
		p := req.Params[0]
		for _, item := range p {
			bs, err := base64.StdEncoding.DecodeString(item.TxnBase64)
			if err != nil {
				return nil, errors.Wrap(err, "failed to base64 decode transaction")
			}

			var txn types.Transaction
			err = msgpack.Decode(bs, &txn)
			if err != nil {
				return nil, errors.Wrap(err, "failed to decode transaction msgpack")
			}

			fmt.Println(ams.FormatTxn(txn))
		}
	}

	// TODO: decode and display transactions
	fmt.Println("Press Enter to sign transactions..")
	s.r.ReadString('\n')

	resp, err := s.s.Sign(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to sign transactions")
	}

	fmt.Println("Signed transactions.")

	return resp, nil
}

func (s *manualConfirmSignerWrapper) Address() string {
	return s.s.Address()
}

func run(a args) error {
	as, err := ams.MakeAddressSource(
		ams.WithAddressString(a.Addr),
		ams.WithAddressThreshold(a.Threshold),
	)
	if err != nil {
		return errors.Wrap(err, "failed to make address source")
	}

	us, err := ams.MakeUriSource(
		ams.WithUriSourceStaticUri(a.Uri),
		ams.WithUriSourceClipboardUri(a.ClipboardUri),
		ams.WithUriSourceNonEmpty(true),
	)
	if err != nil {
		return errors.Wrap(err, "failed to make uri source")
	}

	uri, err := us.Uri()
	if err != nil {
		return errors.Wrap(err, "failed to read uri from source")
	}

	accs, err := ams.MakeAccountSource(
		ams.WithAccountSourceMnemonic(a.Mnemonic),
		ams.WithAccountSourcePrivateKeyPath(a.PrivateKeyPath),
	)
	if err != nil {
		return errors.Wrap(err, "failed to make account source")
	}

	acc, err := accs.ReadAccount()
	if err != nil {
		return errors.Wrap(err, "failed to read account from source")
	}

	signer, err := ams.MakeLocalSigner(as.Address(), acc.PrivateKey,
		ams.WithLocalSignerMatchSender(a.MatchSender),
		ams.WithLocalSignerMultisigAccount(as.Multisig()),
	)
	if err != nil {
		return errors.Wrap(err, "failed to make signer")
	}

	rdr := bufio.NewReader(os.Stdin)

	signer = &manualConfirmSignerWrapper{
		s: signer,
		r: rdr,
	}

	wallet, err := wc.MakeServer(*uri, signer,
		wc.WithServerDebug(a.Debug),
	)
	if err != nil {
		return errors.Wrap(err, "failed to make wallet")
	}

	err = wallet.Run()
	if err != nil {
		return errors.Wrap(err, "failed to run wallet")
	}

	return nil
}

func main() {
	var a args

	flag.StringVar(&a.PrivateKeyPath, "pk-path", "", "private key json file path")

	flag.StringVar(&a.Mnemonic, "mnemonic", "", "private key mnemonic")
	flag.StringVar(&a.Addr, "addr", "", "multisig addresses")

	flag.IntVar(&a.Threshold, "threshold", 0, "multisig threshold")
	flag.StringVar(&a.Txn, "txn", "", "base32 transaction data")
	flag.BoolVar(&a.Debug, "debug", false, "debug mode")
	flag.StringVar(&a.Uri, "uri", "", "WalletConnect uri")
	flag.StringVar(&a.AuthAddr, "auth-addr", "", "Algorand auth address")
	flag.BoolVar(&a.ClipboardUri, "cu", false, "use WalletConnect uri from clipboard")
	flag.StringVar(&a.MatchSender, "match", "", "sign only transactions with matching sender")

	flag.Parse()

	err := run(a)
	if err != nil {
		panic(err)
	}
}
