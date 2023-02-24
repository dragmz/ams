package ams

import (
	"encoding/base64"
	"testing"

	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/encoding/msgpack"
	"github.com/algorand/go-algorand-sdk/transaction"
	"github.com/algorand/go-algorand-sdk/types"
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
	cstx1, err := ConvertToMultisig(stx1, ma)
	assert.NoError(t, err)
	cstx2, err := ConvertToMultisig(stx2, ma)
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

	mstx2, err := ConvertToMultisig(mstx1, ma)
	assert.NoError(t, err)

	assert.Equal(t, mstx1, mstx2)
}

func TestAsastatsGardRegression(t *testing.T) {
	b64s := []string{
		"gqRtc2lng6ZzdWJzaWeVgaJwa8QgGvmMHX6b6H1MT2rL19M1nBP0MEK0h4fm8liEuz6CwbaBonBrxCBsFSS4NaJXEr1FSRK1Zot8GMmMZRfe0WBdA3wfr8FnloGicGvEIEJjHKW5mE3GR9thX/J41LG87f7IBSfNNG8sy2ZsZ9oBgaJwa8QgX3AyFkobhccoQOKAorcXirOuKQm7OozpF4+zJt0N9RKConBrxCCWxnUJxtYKN/ZDeH1WPVDF420qZRauuauAyZYJa1yo2aFzxEDmDocpag7F1MA21xepKxyUb63YS0JsxnoGR+Cf3gn6nacOq7lR34sR0FUGI/bPgC9nNTF//HS6oOm9oIeE1H8Lo3RocgOhdgGjdHhuiqRhYW10zwAACRhOcqAApGFyY3bEIB2KHLXOCSqY7AfPxf59gHeLq6nQDxlJlrR0aGv6Rffzo2ZlZc0D6KJmds4BnwvKomdoxCDAYcTY/B293tLXYEvkVo4/bQQZh6w3veS2ILWrOSSK36Jsds4Bnw+ypG5vdGXECG8NyRxl57oPo3NuZMQgkpuYAtk/P7pdBm4XMlyB3nsdvoNIioziZJV9nbXLyRCkdHlwZaVheGZlcqR4YWlkzhd06Ic=",
		"gqRtc2lng6ZzdWJzaWeVgaJwa8QgGvmMHX6b6H1MT2rL19M1nBP0MEK0h4fm8liEuz6CwbaBonBrxCBsFSS4NaJXEr1FSRK1Zot8GMmMZRfe0WBdA3wfr8FnloKicGvEIEJjHKW5mE3GR9thX/J41LG87f7IBSfNNG8sy2ZsZ9oBoXPEQKSQpE2oc9N94UsLLx3JopfGI+0t1hnJNX1QWw5oBawm1Mnq4HAcZ6AlWIARhBCt8m9EEFEMv244yRgYXCCqLAiBonBrxCBfcDIWShuFxyhA4oCitxeKs64pCbs6jOkXj7Mm3Q31EoGicGvEIJbGdQnG1go39kN4fVY9UMXjbSplFq65q4DJlglrXKjZo3RocgOhdgGjdHhuiqRhYW10zwAACRhOcqAApGFyY3bEIB2KHLXOCSqY7AfPxf59gHeLq6nQDxlJlrR0aGv6Rffzo2ZlZc0D6KJmds4BnwvKomdoxCDAYcTY/B293tLXYEvkVo4/bQQZh6w3veS2ILWrOSSK36Jsds4Bnw+ypG5vdGXECG8NyRxl57oPo3NuZMQgkpuYAtk/P7pdBm4XMlyB3nsdvoNIioziZJV9nbXLyRCkdHlwZaVheGZlcqR4YWlkzhd06Ic=",
		"gqRtc2lng6ZzdWJzaWeVgqJwa8QgGvmMHX6b6H1MT2rL19M1nBP0MEK0h4fm8liEuz6Cwbahc8RAUiQKOfOI1BU5K3fhEqLQfUPwmiisErEiMzpNf3LC+b+kV4yKRznbJeCcjqSpbfhlzWzuMZlw2sABifZ9EGsxBoGicGvEIGwVJLg1olcSvUVJErVmi3wYyYxlF97RYF0DfB+vwWeWgaJwa8QgQmMcpbmYTcZH22Ff8njUsbzt/sgFJ800byzLZmxn2gGBonBrxCBfcDIWShuFxyhA4oCitxeKs64pCbs6jOkXj7Mm3Q31EoGicGvEIJbGdQnG1go39kN4fVY9UMXjbSplFq65q4DJlglrXKjZo3RocgOhdgGjdHhuiqRhYW10zwAACRhOcqAApGFyY3bEIB2KHLXOCSqY7AfPxf59gHeLq6nQDxlJlrR0aGv6Rffzo2ZlZc0D6KJmds4BnwvKomdoxCDAYcTY/B293tLXYEvkVo4/bQQZh6w3veS2ILWrOSSK36Jsds4Bnw+ypG5vdGXECG8NyRxl57oPo3NuZMQgkpuYAtk/P7pdBm4XMlyB3nsdvoNIioziZJV9nbXLyRCkdHlwZaVheGZlcqR4YWlkzhd06Ic=",
	}

	astrs := []string{"DL4YYHL6TPUH2TCPNLF5PUZVTQJ7IMCCWSDYPZXSLCCLWPUCYG3DI7CA7I", "NQKSJOBVUJLRFPKFJEJLKZULPQMMTDDFC7PNCYC5AN6B7L6BM6LLI4F5QM", "IJRRZJNZTBG4MR63MFP7E6GUWG6O37WIAUT42NDPFTFWM3DH3IA5TJ4QGA", "L5YDEFSKDOC4OKCA4KAKFNYXRKZ24KIJXM5IZ2IXR6ZSNXIN6UJD2AZ7OI", "S3DHKCOG2YFDP5SDPB6VMPKQYXRW2KTFC2XLTK4AZGLAS224VDMUAXO5UQ"}
	var addrs []types.Address

	for _, astr := range astrs {
		addr, err := types.DecodeAddress(astr)
		assert.NoError(t, err)

		addrs = append(addrs, addr)
	}

	ma, err := crypto.MultisigAccountWithParams(1, 3, addrs)
	assert.NoError(t, err)

	//maddr, err := ma.Address()
	assert.NoError(t, err)

	var mbss [][]byte

	for _, b64 := range b64s {
		bs, err := base64.StdEncoding.DecodeString(b64)
		assert.NoError(t, err)

		var stx types.SignedTxn
		err = msgpack.Decode(bs, &stx)
		assert.NoError(t, err)

		assert.Equal(t, types.ZeroAddress, stx.AuthAddr)

		mbs, err := ConvertToMultisig(bs, ma)
		assert.NoError(t, err)

		mbss = append(mbss, mbs)
	}

	id, _, err := crypto.MergeMultisigTransactions(mbss...)
	assert.NoError(t, err)

	assert.Equal(t, "BQHE7KAK34WIXOM2TK7IFAU7YTAU3XRMHAH666ON4UUGDQGYJ4TQ", id)
}
