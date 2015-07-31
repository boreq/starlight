package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"errors"
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

func (k rsaPrivateKey) PublicKey() PublicKey {
	return rsaPublicKey{&k.key.PublicKey}
}

func NewPrivateKey(data []byte) (PrivateKey, error) {
	key, err := x509.ParsePKCS1PrivateKey(data)
	if err != nil {
		return nil, err
	}
	return rsaPrivateKey{key}, nil
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

func NewPublicKey(data []byte) (PublicKey, error) {
	decodedKey, err := x509.ParsePKIXPublicKey(data)
	if err != nil {
		return nil, err
	}
	key, ok := decodedKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("This is not a RSA key.")
	}
	return rsaPublicKey{key}, nil
}

// Generate an RSA keypair of the specified length.
func GenerateKeypair(bits int) (PrivateKey, PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, err
	}
	return rsaPrivateKey{privateKey}, rsaPublicKey{&privateKey.PublicKey}, nil
}
