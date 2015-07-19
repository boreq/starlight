package crypto

import "encoding/hex"

type Key interface {
	Bytes() ([]byte, error)
	Hash() ([]byte, error)
}

type PrivateKey interface {
	Key
}

type PublicKey interface {
	Key
}

func EncodeHex(data []byte) string {
	return hex.EncodeToString(data)
}
