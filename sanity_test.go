package ams_test

import (
	"context"
	"testing"

	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/transaction"
	"github.com/algorand/go-algorand-sdk/types"
	"github.com/stretchr/testify/assert"
)

func TestMultisigAuthAddr(t *testing.T) {
	ac, err := algod.MakeClient("https://testnet-api.algonode.network", "")
	assert.NoError(t, err)

	var accs []crypto.Account
	var addrs []types.Address

	for i := 0; i < 4; i++ {
		acc := crypto.GenerateAccount()
		accs = append(accs, acc)
		addrs = append(addrs, acc.Address)
	}

	ma, err := crypto.MultisigAccountWithParams(1, 3, addrs)
	assert.NoError(t, err)

	maddr, err := ma.Address()
	assert.NoError(t, err)

	sp, err := ac.SuggestedParams().Do(context.Background())
	assert.NoError(t, err)

	tx, err := transaction.MakePaymentTxn(maddr.String(), maddr.String(), transaction.MinTxnFee, 0, uint64(sp.FirstRoundValid), uint64(sp.LastRoundValid), nil, "", sp.GenesisID, sp.GenesisHash)
	assert.NoError(t, err)

	for _, acc := range accs {
		_, _, err = crypto.SignMultisigTransaction(acc.PrivateKey, ma, tx)
		assert.NoError(t, err)
	}

}
