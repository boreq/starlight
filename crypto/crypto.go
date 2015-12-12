package crypto

import (
	"crypto"
	"crypto/rand"
)

type Key interface {
	Bytes() ([]byte, error)
	Hash() ([]byte, error)
}

type PrivateKey interface {
	Key

	// PublicKey returns the underlying public key.
	PublicKey() PublicKey

	// Sign signs the provided data using a given hash function.
	Sign(data []byte, hash crypto.Hash) ([]byte, error)
}

type PublicKey interface {
	Key

	// Validate validates a signature using a given hash function.
	Validate(data, signature []byte, hash crypto.Hash) error
}

type EphemeralKey interface {
	// Bytes returns the public key bytes in a format suitable to be used by
	// GenerateSharedSecret method.
	Bytes() ([]byte, error)

	// GenerateSharedSecret generates a shared secret using the public key
	// bytes received from the second party.
	GenerateSharedSecret([]byte) ([]byte, error)
}

// Must be variables or there will be a problem with using those in the protobuf
// structs.
var SupportedCurves = "P224,P256,P384,P521"
var SupportedHashes = "SHA256,SHA512"
var SupportedCiphers = "AES-256,AES-128"

// GenerateNonce fills a provided slice with random bytes.
func GenerateNonce(nonce []byte) error {
	_, err := rand.Read(nonce)
	return err
}
