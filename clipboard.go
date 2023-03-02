package ams

import (
	"github.com/atotto/clipboard"
	"github.com/pkg/errors"
)

func ReadWcFromClipboard() (string, error) {
	str, err := clipboard.ReadAll()
	if err != nil {
		return "", errors.Wrap(err, "failed to read from clipboard")
	}

	return str, nil
}
