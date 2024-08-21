package encryption

import (
	"crypto/rand"
	"crypto/rsa"
)

func Encrypt(privateKey *rsa.PrivateKey, data []byte) ([]byte, error) {
	return rsa.EncryptPKCS1v15(
		rand.Reader,
		&privateKey.PublicKey,
		data,
	)
}

func Decrypt(privateKey *rsa.PrivateKey, encrypted []byte) ([]byte, error) {
	return rsa.DecryptPKCS1v15(
		rand.Reader,
		privateKey,
		encrypted,
	)
}
