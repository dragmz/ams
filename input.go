package ams

import (
	"bufio"
	"os"
	"strings"

	"github.com/pkg/errors"
)

func ReadInput(rdr *bufio.Reader) (string, error) {
	input, err := rdr.ReadString('\n')
	if err != nil {
		return "", errors.Wrap(err, "failed to read transaction from reader")
	}

	input = strings.TrimSpace(input)
	if strings.HasPrefix(input, "<") {
		bs, err := os.ReadFile(input[1:])
		if err != nil {
			return "", errors.Wrap(err, "failed to read transaction file")
		}

		input = string(bs)
	}

	return input, nil
}
