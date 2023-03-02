package ams

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"

	"github.com/pkg/errors"
	"golang.org/x/crypto/pbkdf2"
)

type KeyCryptoConfig struct {
	Salt       []byte      `json:"salt"`
	Iterations int         `json:"iterations"`
	KeyLength  int         `json:"key_length"`
	Hash       crypto.Hash `json:"hash"`
	Iv         []byte      `json:"iv"`
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

	e := cipher.NewCBCEncrypter(c, kc.Iv)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make cbc")
	}

	cbs := make([]byte, len(plainBytes))
	e.CryptBlocks(cbs, plainBytes)

	return cbs, nil
}

func (kc KeyCryptoConfig) Decrypt(cipherBytes []byte, password string) ([]byte, error) {
	c, err := kc.makeCipher(password)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make cipher")
	}

	e := cipher.NewCBCDecrypter(c, kc.Iv)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make cbc")
	}

	pbs := make([]byte, len(cipherBytes))
	e.CryptBlocks(pbs, cipherBytes)

	return pbs, nil
}
