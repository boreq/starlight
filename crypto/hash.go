package crypto

import "crypto/sha256"

func Digest(data []byte) []byte {
	d := sha256.Sum256(data)
	return d[:]
}

func KeyDigest(key Key) ([]byte, error) {
	b, err := key.Bytes()
	if err != nil {
		return nil, err
	}
	return Digest(b), nil
}
