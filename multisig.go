package ams

import (
	"bytes"
	"crypto/ed25519"

	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/encoding/msgpack"
	"github.com/algorand/go-algorand-sdk/types"
	"github.com/pkg/errors"
)

// Service function to make a single signature in Multisig
func multisigSingle(pk ed25519.PublicKey, ma crypto.MultisigAccount, rawSig types.Signature) (msig types.MultisigSig, myIndex int, err error) {
	myIndex = len(ma.Pks)
	myPublicKey := pk
	for i := 0; i < len(ma.Pks); i++ {
		if bytes.Equal(myPublicKey, ma.Pks[i]) {
			myIndex = i
		}
	}
	if myIndex == len(ma.Pks) {
		err = errors.New("errMsigInvalidSecretKey")
		return
	}

	// now, create the signed transaction
	msig.Version = ma.Version
	msig.Threshold = ma.Threshold
	msig.Subsigs = make([]types.MultisigSubsig, len(ma.Pks))
	for i := 0; i < len(ma.Pks); i++ {
		c := make([]byte, len(ma.Pks[i]))
		copy(c, ma.Pks[i])
		msig.Subsigs[i].Key = c
	}

	msig.Subsigs[myIndex].Sig = rawSig
	return
}

func ConvertToMultisig(bs []byte, ma crypto.MultisigAccount) ([]byte, error) {
	var stx types.SignedTxn

	err := msgpack.Decode(bs, &stx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode signed transaction msgpack")
	}

	if !stx.Msig.Blank() {
		return bs, nil
	}

	var signer types.Address

	if !stx.AuthAddr.IsZero() {
		signer = stx.AuthAddr
	} else {
		signer = stx.Txn.Sender
	}

	pk := make([]byte, ed25519.PublicKeySize)

	copy(pk[:], signer[:])

	sig, _, err := multisigSingle(pk, ma, stx.Sig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to multisig single")
	}

	smstx := types.SignedTxn{
		Msig: sig,
		Txn:  stx.Txn,
	}

	maAddress, err := ma.Address()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get multisig address")
	}

	if smstx.Txn.Sender != maAddress {
		smstx.AuthAddr = maAddress
	}

	mstx := msgpack.Encode(smstx)

	return mstx, nil
}
