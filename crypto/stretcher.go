package crypto

import (
	"crypto/aes"
	"errors"
	"golang.org/x/crypto/pbkdf2"
)

type StretchedKeys struct {
	IV        []byte
	MacKey    []byte
	CipherKey []byte
}

// Looks like mac key size is arbitrary.
const macKeySize = 30

// StretchedKey creates two new keypairs from the shared secret.
func StretchKey(secret, salt []byte, hashName, cipherName string) (a, b StretchedKeys, err error) {
	// We want to generate 2 * (IV, mac key, cipher key) so we need to know
	// the total number of bytes we need.

	// IV length is equal to cipher block size.
	var keySize, blockSize int
	switch cipherName {
	case "AES-256":
		keySize = 32
		blockSize = aes.BlockSize
	case "AES-128":
		keySize = 16
		blockSize = aes.BlockSize
	default:
		err = errors.New("Invalid cipher name")
		return
	}

	// Hash.
	h, err := GetCryptoHash(hashName)
	if err != nil {
		return
	}

	totalSize := 2 * (keySize + blockSize + macKeySize)
	key := pbkdf2.Key(secret, salt, 4096, totalSize, h.New)

	keyA := key[:totalSize/2]
	a.IV = keyA[:blockSize]
	a.MacKey = keyA[blockSize : blockSize+macKeySize]
	a.CipherKey = keyA[blockSize+macKeySize:]

	keyB := key[totalSize/2:]
	b.IV = keyB[:blockSize]
	b.MacKey = keyB[blockSize : blockSize+macKeySize]
	b.CipherKey = keyB[blockSize+macKeySize:]

	return
}
