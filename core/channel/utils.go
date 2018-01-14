package channel

import (
	"crypto/sha256"
	"github.com/boreq/starlight/crypto"
)

// idHash is a hash used to encode channel names in the DHT.
var idHash = sha256.New()

// IdLength is a length of a channel id produced by the CreateId function.
var IdLength = idHash.Size()

// CreateId returns an id used in DHT for the given channel name.
func CreateId(name string) []byte {
	return crypto.Digest(idHash, []byte(name))
}

// ValidateId returns true if the given channel id is valid.
func ValidateId(id []byte) bool {
	if len(id) != IdLength {
		return false
	}
	return true
}
