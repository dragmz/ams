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
	"github.com/dragmz/wc"
	"github.com/pkg/errors"
)

type args struct {
	Mnemonic  string
	Addr      string
	Threshold int
	Txn       string
	Debug     bool
	Uri       string
	AuthAddr  string

	ClipboardQrUri bool
	ClipboardUri   bool
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
	addrs, err := ams.ParseAddrs(a.Addr, ",")
	if err != nil {
		return errors.Wrap(err, "failed to parse addresses")
	}

	if len(addrs) == 0 {
		return errors.New("missing address")
	}

	var addr string

	var ma crypto.MultisigAccount
	if len(addrs) > 1 {
		ma, err = crypto.MultisigAccountWithParams(1, uint8(a.Threshold), addrs)
		if err != nil {
			return errors.Wrap(err, "failed to build multisig account")
		}

		maddr, err := ma.Address()
		if err != nil {
			return errors.Wrap(err, "failed to get multisig address")
		}

		fmt.Println("Multisig address:", maddr)
		addr = maddr.String()
	} else {
		fmt.Println("Address:", addrs[0])
		addr = addrs[0].String()
	}

	sk, err := mnemonic.ToPrivateKey(a.Mnemonic)
	if err != nil {
		return errors.Wrap(err, "failed to convert mnemonic to private key")
	}

	acc, err := crypto.AccountFromPrivateKey(sk)
	if err != nil {
		return errors.Wrap(err, "failed to convert private key to account")
	}

	rdr := bufio.NewReader(os.Stdin)

	uris := 0
	if len(a.Uri) > 0 {
		uris++
	}
	if a.ClipboardQrUri {
		uris++
	}
	if a.ClipboardUri {
		uris++
	}

	if uris > 1 {
		return errors.New("only one uri can be used")
	}

	uristr := a.Uri

	if len(uristr) == 0 && a.ClipboardQrUri {
		uristr, err = ams.ReadQrFromClipboard()
		if err != nil {
			return errors.Wrap(err, "failed to read uri from clipboard qr code")
		}
	}

	if len(uristr) == 0 && a.ClipboardUri {
		uristr, err = ams.ReadWcFromClipboard()
		if err != nil {
			return errors.Wrap(err, "Failed to read uri from clipboard")
		}
	}

	uri, err := wc.ParseUri(uristr)
	if err != nil {
		return errors.Wrap(err, "failed to parse uri")
	}

	sopts := []ams.LocalSignerOption{}

	if len(addrs) > 1 {
		sopts = append(sopts,
			ams.WithLocalSignerMultisigAccount(ma))
	}

	signer, err := ams.MakeLocalSigner(addr, acc.PrivateKey, sopts...)
	if err != nil {
		return errors.Wrap(err, "failed to make signer")
	}

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

	flag.StringVar(&a.Mnemonic, "mnemonic", "", "private key mnemonic")
	flag.StringVar(&a.Addr, "addr", "", "multisig addresses")
	flag.IntVar(&a.Threshold, "threshold", 0, "multisig threshold")
	flag.StringVar(&a.Txn, "txn", "", "base32 transaction data")
	flag.BoolVar(&a.Debug, "debug", false, "debug mode")
	flag.StringVar(&a.Uri, "uri", "", "WalletConnect uri")
	flag.StringVar(&a.AuthAddr, "auth-addr", "", "Algorand auth address")
	flag.BoolVar(&a.ClipboardQrUri, "cqu", false, "use WalletConnect uri from QR code in clipboard")
	flag.BoolVar(&a.ClipboardUri, "cu", false, "use WalletConnect uri from clipboard")

	flag.Parse()

	err := run(a)
	if err != nil {
		panic(err)
	}
}
