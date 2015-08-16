package crypto

type Key interface {
	Bytes() ([]byte, error)
	Hash() ([]byte, error)
}

type PrivateKey interface {
	Key
	PublicKey() PublicKey
}

type PublicKey interface {
	Key
}

type EphemeralKey interface {
	// Bytes returns the public key bytes in a format suitable to be used by
	// GenerateSharedSecret method.
	Bytes() ([]byte, error)

	// GenerateSharedSecret generates a shared secret using the public key
	// bytes received from the second party.
	GenerateSharedSecret([]byte) ([]byte, error)
}
