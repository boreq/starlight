package node

import (
	"bytes"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"github.com/boreq/lainnet/crypto"
	"github.com/boreq/lainnet/utils"
	"io/ioutil"
	"path"
)

type ID []byte

func (id ID) String() string {
	return hex.EncodeToString(id)
}

func (id *ID) MarshalJSON() ([]byte, error) {
	buf := bytes.Buffer{}
	buf.WriteRune('"')
	buf.WriteString(id.String())
	buf.WriteRune('"')
	return buf.Bytes(), nil
}

func (id *ID) UnmarshalJSON(data []byte) error {
	decId, err := hex.DecodeString(string(data[1 : len(data)-1]))
	if err != nil {
		return err
	}
	// I don't even...
	newId := ID(decId)
	*id = newId
	return nil
}

func NewId(id string) (ID, error) {
	return hex.DecodeString(id)
}

type Identity struct {
	Id      ID
	PubKey  crypto.PublicKey
	PrivKey crypto.PrivateKey
}

type NodeInfo struct {
	Id      ID
	Address string
}

const minKeyBits = 2048

// CompareId returns true if two IDs are exactly the same.
func CompareId(a, b ID) bool {
	return bytes.Compare(a, b) == 0
}

// Distance calculates the distance between two nodes.
func Distance(a, b ID) ([]byte, error) {
	// XOR is the distance metric, to actually get a meaningful distance
	// from it we just count the preceeding zeros.
	return utils.XOR(a, b)
}

// ValidateId returns true if a node id is valid - has proper length and proper
// structure (correct length of a prefix consisting of zero bits).
func ValidateId(id ID) bool {
	if len(id) != crypto.KeyDigestLength {
		return false
	}
	// TODO implement the prefix checks
	return true
}

// Generates a fresh identity (keypair) for a local node.
func GenerateIdentity(bits, difficulty int) (*Identity, error) {
	// Should this be placed here and not in the init command?
	if bits < minKeyBits {
		return nil, fmt.Errorf("Use at least %d bits to generate a key", minKeyBits)
	}

	privKey, pubKey, err := crypto.GenerateKeypair(bits)
	if err != nil {
		return nil, err
	}

	id, err := pubKey.Hash()
	if err != nil {
		return nil, err
	}

	return &Identity{id, pubKey, privKey}, nil
}

const identityFilename = "identity.pem"

// SaveLocalIdentity saves the local identity in the specified directory.
func SaveLocalIdentity(iden *Identity, directory string) error {
	path := path.Join(directory, identityFilename)
	keyBytes, err := iden.PrivKey.Bytes()
	if err != nil {
		return err
	}
	data := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: keyBytes,
		},
	)
	err = ioutil.WriteFile(path, data, 0600)
	if err != nil {
		return err
	}
	return nil
}

// LoadLocalIdentity loads the local identity from the specified directory.
func LoadLocalIdentity(directory string) (*Identity, error) {
	path := path.Join(directory, identityFilename)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	privKey, err := crypto.NewPrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	pubKey := privKey.PublicKey()
	hash, err := pubKey.Hash()
	if err != nil {
		return nil, err
	}

	iden := Identity{
		PrivKey: privKey,
		PubKey:  pubKey,
		Id:      hash,
	}
	return &iden, nil
}
