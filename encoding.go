package ams

import "encoding/base32"

var TxnTransferEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)
