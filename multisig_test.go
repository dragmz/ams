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
	acc1 := crypto.GenerateAccount()
	acc2 := crypto.GenerateAccount()

	ma, err := crypto.MultisigAccountWithParams(1, 2, []types.Address{acc1.Address, acc2.Address})
	assert.NoError(t, err)

	maddr, err := ma.Address()
	assert.NoError(t, err)

	tx, err := transaction.MakePaymentTxn(maddr.String(), maddr.String(), 1000, 123, 1000, 2000, nil, "", "test", []byte("test"))
	assert.NoError(t, err)

	// Sign as single, then convert to multisig and merge
	_, stx1, err := crypto.SignTransaction(acc1.PrivateKey, tx)
	assert.NoError(t, err)
	_, stx2, err := crypto.SignTransaction(acc2.PrivateKey, tx)
	assert.NoError(t, err)
	cstx1, err := ams.ConvertToMultisig(stx1, ma)
	assert.NoError(t, err)
	cstx2, err := ams.ConvertToMultisig(stx2, ma)
	assert.NoError(t, err)
	_, a, err := crypto.MergeMultisigTransactions(cstx1, cstx2)
	assert.NoError(t, err)

	// Sign as multisig, then merge
	_, mstx1, err := crypto.SignMultisigTransaction(acc1.PrivateKey, ma, tx)
	assert.NoError(t, err)
	_, mstx2, err := crypto.SignMultisigTransaction(acc2.PrivateKey, ma, tx)
	assert.NoError(t, err)
	_, b, err := crypto.MergeMultisigTransactions(mstx1, mstx2)
	assert.NoError(t, err)

	assert.Equal(t, a, b)
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
