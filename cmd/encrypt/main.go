package main

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/algorand/go-algorand-sdk/mnemonic"
	"github.com/dragmz/ams"
	"github.com/pkg/errors"
	"golang.org/x/term"
)

type args struct {
	Mnemonic   string
	Password   string
	Output     string
	Iterations int
	SaltLength int
}

func run(a args) error {
	if len(a.Password) == 0 {
		fmt.Println("Enter password:")
		bs, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return errors.Wrap(err, "failed to read password")
		}

		a.Password = string(bs)
		fmt.Println(a.Password)
	}
	sk, err := mnemonic.ToPrivateKey(a.Mnemonic)
	if err != nil {
		return errors.Wrap(err, "failed to convert mnemonic to private key")
	}

	salt := make([]byte, a.SaltLength)
	_, err = rand.Read(salt)
	if err != nil {
		return errors.Wrap(err, "failed to generate salt")
	}

	iv := make([]byte, 16)
	_, err = rand.Read(iv)
	if err != nil {
		return errors.Wrap(err, "failed to make iv")
	}

	iterations := a.Iterations
	keyLength := 32
	hash := crypto.SHA256

	kc := ams.KeyCryptoConfig{
		Salt:       salt,
		Iterations: iterations,
		KeyLength:  keyLength,
		Hash:       hash,
		Iv:         iv,
	}

	cipher, err := kc.Encrypt(sk, a.Password)
	if err != nil {
		return errors.Wrap(err, "failed to encrypt private key")
	}

	actualPlain, err := kc.Decrypt(cipher, a.Password)
	if !bytes.Equal(actualPlain, sk) {
		return errors.Wrap(err, "failed to verify")
	}

	f, err := os.Create(a.Output)
	if err != nil {
		return errors.Wrap(err, "failed to create file")
	}

	defer f.Close()

	je := json.NewEncoder(f)
	err = je.Encode(ams.KeyCryptoPackage{
		Config: kc,
		Cipher: cipher,
	})

	if err != nil {
		return errors.Wrap(err, "failed to encode")
	}

	return nil
}

func main() {
	var a args
	flag.StringVar(&a.Mnemonic, "mnemonic", "", "mnemonic to encrypt")
	flag.StringVar(&a.Password, "password", "", "password")
	flag.StringVar(&a.Output, "output", "", "output file")
	flag.IntVar(&a.Iterations, "iterations", 1024*1024, "number of pbkdf2 iterations")
	flag.IntVar(&a.SaltLength, "salt-length", 32, "pbkdf2 salt length")
	flag.Parse()

	err := run(a)
	if err != nil {
		panic(err)
	}
}
