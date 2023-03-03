package ams

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"os"

	algocrypto "github.com/algorand/go-algorand-sdk/crypto"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/pbkdf2"
)

type KeyCryptoConfig struct {
	Salt       []byte      `json:"salt"`
	Iterations int         `json:"iterations"`
	KeyLength  int         `json:"key_length"`
	Hash       crypto.Hash `json:"hash"`
	Nonce      []byte      `json:"nonce"`
}

type KeyCryptoPackage struct {
	Config KeyCryptoConfig `json:"config"`
	Cipher []byte          `json:"cipher"`
}

func (p KeyCryptoPackage) Decrypt(password string) ([]byte, error) {
	return p.Config.Decrypt(p.Cipher, password)
}

func (kc KeyCryptoConfig) makeCipher(password string) (cipher.Block, error) {
	dk := pbkdf2.Key([]byte(password), kc.Salt, kc.Iterations, kc.KeyLength, kc.Hash.New)

	c, err := aes.NewCipher(dk)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make cipher")
	}

	return c, nil
}

func (kc KeyCryptoConfig) Encrypt(plainBytes []byte, password string) ([]byte, error) {
	c, err := kc.makeCipher(password)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make cipher")
	}

	e, err := cipher.NewGCM(c)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make gcm")
	}

	cbs := e.Seal(nil, kc.Nonce, plainBytes, nil)

	return cbs, nil
}

func (kc KeyCryptoConfig) Decrypt(cipherBytes []byte, password string) ([]byte, error) {
	c, err := kc.makeCipher(password)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make cipher")
	}

	e, err := cipher.NewGCM(c)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make cbc")
	}

	pbs, err := e.Open(nil, kc.Nonce, cipherBytes, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decrypt")
	}

	return pbs, nil
}

func ReadAccountFromFile(path string, password string) (*algocrypto.Account, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open file")
	}

	r := json.NewDecoder(f)

	var kp KeyCryptoPackage
	err = r.Decode(&kp)
	if err != nil {
		return nil, errors.Wrap(err, "faild to read key crypto package")
	}

	bs, err := kp.Decrypt(password)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decrypt key")
	}

	sk := ed25519.PrivateKey(bs)

	acc, err := algocrypto.AccountFromPrivateKey(sk)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read account from private key")
	}

	return &acc, nil
}
