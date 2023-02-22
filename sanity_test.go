package ams_test

import (
	"testing"

	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/transaction"
	"github.com/algorand/go-algorand-sdk/types"
	"github.com/dragmz/ams"
	"github.com/stretchr/testify/assert"
)

func TestMultiSigFromSig(t *testing.T) {
	acc := crypto.GenerateAccount()

	tx, err := transaction.MakePaymentTxn(acc.Address.String(), acc.Address.String(), 1000, 123, 1000, 2000, nil, "", "test", []byte("test"))
	assert.NoError(t, err)

	ma, err := crypto.MultisigAccountWithParams(1, 1, []types.Address{acc.Address})
	assert.NoError(t, err)

	_, stx1, err := crypto.SignTransaction(acc.PrivateKey, tx)
	assert.NoError(t, err)
	mstx1, err := ams.ConvertToMultisig(stx1, ma)
	assert.NoError(t, err)

	_, mstx2, err := crypto.SignMultisigTransaction(acc.PrivateKey, ma, tx)
	assert.NoError(t, err)

	assert.Equal(t, mstx1, mstx2)
}

func TestMultiSigFromMultiSig(t *testing.T) {
	acc := crypto.GenerateAccount()

	tx, err := transaction.MakePaymentTxn(acc.Address.String(), acc.Address.String(), 1000, 123, 1000, 2000, nil, "", "test", []byte("test"))
	assert.NoError(t, err)

	ma, err := crypto.MultisigAccountWithParams(1, 1, []types.Address{acc.Address})
	assert.NoError(t, err)

	_, mstx1, err := crypto.SignMultisigTransaction(acc.PrivateKey, ma, tx)
	assert.NoError(t, err)

	mstx2, err := ams.ConvertToMultisig(mstx1, ma)
	assert.NoError(t, err)

	assert.Equal(t, mstx1, mstx2)
}
