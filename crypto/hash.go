package crypto

import (
	"crypto"
	"crypto/sha256"
	"errors"
	"hash"
)

// Digest calculates a checksum using the provided hash.
func Digest(hash hash.Hash, data []byte) []byte {
	var sum []byte
	hash.Reset()
	hash.Write(data)
	return hash.Sum(sum)
}

// keyDigestHash is a hash used by the KeyDigest function.
var keyDigestHash = sha256.New()

// KeyDigestLength is a length of a key digest produced by the KeyDigest
// function.
var KeyDigestLength = keyDigestHash.Size()

// KeyDigest calculates a SHA256 checksum of a key.
func KeyDigest(key Key) ([]byte, error) {
	b, err := key.Bytes()
	if err != nil {
		return nil, err
	}
	return Digest(keyDigestHash, b), nil
}

// GetHash returns a hash.Hash based on the name.
func GetHash(name string) (hash.Hash, error) {
	f, err := GetCryptoHash(name)
	if err != nil {
		return nil, err
	}
	return f.New(), nil
}

// GetCryptoHash returns a crypto.Hash based on the name.
func GetCryptoHash(name string) (hash crypto.Hash, err error) {
	switch name {
	case "SHA256":
		hash = crypto.SHA256
	case "SHA512":
		hash = crypto.SHA512
	default:
		err = errors.New("Invalid hash name")
		return
	}

	if !hash.Available() {
		err = errors.New("Hash is not available")
	}
	return
}
