package node

import (
	"fmt"
	"github.com/boreq/netblog/crypto"
	"github.com/boreq/netblog/encode"
	"github.com/boreq/netblog/utils"
	"io/ioutil"
	"path"
)

const minKeyBits = 2048

type Identity struct {
	Id      []byte
	PubKey  crypto.PublicKey
	PrivKey crypto.PrivateKey
}

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

func saveLocalKey(key crypto.Key, path string) error {
	b, err := key.Bytes()
	if err != nil {
		return err
	}
	b64 := encode.Base64Encode(b)
	err = ioutil.WriteFile(path, b64, 0600)
	if err != nil {
		return err
	}
	return nil
}

// Saves local identity keys in directory/{private_key,public_key}.
func SaveLocalIdentity(iden *Identity, directory string) error {
	if err := utils.EnsureDirExists(directory, false); err != nil {
		return err
	}

	privPath := path.Join(directory, "private_key")
	if err := saveLocalKey(iden.PrivKey, privPath); err != nil {
		return err
	}

	pubPath := path.Join(directory, "public_key")
	if err := saveLocalKey(iden.PubKey, pubPath); err != nil {
		return err
	}

	return nil
}
