package ams

import (
	"fmt"
	"strings"

	"github.com/algorand/go-algorand-sdk/types"
)

func FormatTxn(txn types.Transaction) string {
	out := strings.Builder{}

	out.WriteString(fmt.Sprintf("Type: %s\n", txn.Type))
	out.WriteString(fmt.Sprintf("Sender: %s\n", txn.Sender))
	out.WriteString(fmt.Sprintf("Group Id: %s\n", txn.Group))
	out.WriteString(fmt.Sprintf("Validity: %d..%d (%d rounds)\n", txn.FirstValid, txn.LastValid, txn.LastValid-txn.FirstValid+1))

	if !txn.RekeyTo.IsZero() {
		out.WriteString(fmt.Sprintf("[!!!] REKEY TO: %s\n", txn.RekeyTo.String()))
	}

	switch txn.Type {
	case types.ApplicationCallTx:
		out.WriteString(fmt.Sprintf("Application ID: %d\n", txn.ApplicationID))
		out.WriteString(fmt.Sprintf("On Complete: %d\n", txn.OnCompletion))
	}

	out.WriteString(fmt.Sprintf("Fee: %d microALGO\n", txn.Fee))
	if len(txn.Note) > 0 {
		out.WriteString(fmt.Sprintf("Note: %s\n", txn.Note))
	}

	return out.String()
}
