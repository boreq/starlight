package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
)

// Implements PrivateKey.
type rsaPrivateKey struct {
	key *rsa.PrivateKey
}

func (k rsaPrivateKey) Bytes() ([]byte, error) {
	return x509.MarshalPKCS1PrivateKey(k.key), nil
}

func (k rsaPrivateKey) Hash() ([]byte, error) {
	return KeyDigest(k)
}

// Implements PublicKey.
type rsaPublicKey struct {
	key *rsa.PublicKey
}

func (k rsaPublicKey) Bytes() ([]byte, error) {
	return x509.MarshalPKIXPublicKey(k.key)
}

func (k rsaPublicKey) Hash() ([]byte, error) {
	return KeyDigest(k)
}

// Generate an RSA keypair of the specified length.
func GenerateKeypair(bits int) (PrivateKey, PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, err
	}
	return rsaPrivateKey{privateKey}, rsaPublicKey{&privateKey.PublicKey}, nil
}
