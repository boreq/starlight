package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
)

// GetCipher returns a block cipher based on the given name.
func GetCipher(name string, key []byte) (cipher.Block, error) {
	switch name {
	case "AES-256":
		if len(key) != 32 {
			return nil, errors.New("Key must be 32 bytes long")
		}
		return aes.NewCipher(key)
	case "AES-128":
		if len(key) != 16 {
			return nil, errors.New("Key must be 16 bytes long")
		}
		return aes.NewCipher(key)
	default:
		return nil, errors.New("Invalid cipher name")
	}
}
