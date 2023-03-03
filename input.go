package ams

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/term"
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

func ReadPasswordFromStdin(confirm bool) (string, error) {
	fmt.Println("Enter password:")
	bs, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", errors.Wrap(err, "failed to read password")
	}

	if confirm {
		fmt.Println("Enter password again:")
		bs2, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return "", errors.Wrap(err, "failed to read password again")
		}

		if !bytes.Equal(bs, bs2) {
			return "", errors.Wrap(err, "passwords do not match")
		}
	}

	return string(bs), nil
}
