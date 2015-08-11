package node

import (
	"fmt"
	"github.com/boreq/netblog/crypto"
	"testing"
)

func TestPublic(t *testing.T) {
	iden, err := GenerateIdentity(2048, 0)
	if err != nil {
		t.Fatal(err)
	}

	b, err := iden.PubKey.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	pub, err := crypto.NewPublicKey(b)
	if err != nil {
		t.Fatal(err)
	}

	h, err := pub.Hash()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%x\n", h)
}
