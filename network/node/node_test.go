package node

import (
	"fmt"
	"github.com/boreq/lainnet/crypto"
	"testing"
)

func TestPublic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}

	const numBits = 2048

	iden, err := GenerateIdentity(numBits)
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

func TestValidate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}

	const numBits = 2048

	iden, err := GenerateIdentity(numBits)
	if err != nil {
		t.Fatal(err)
	}

	// Correct.
	if !ValidateId(iden.Id) {
		t.Fatal("Id validation failed")
	}
}

func BenchmarkGenerateIdentity2048(b *testing.B) {
	const numBits = 2048

	for i := 0; i < b.N; i++ {
		_, err := GenerateIdentity(numBits)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerateIdentity4096(b *testing.B) {
	const numBits = 4096

	for i := 0; i < b.N; i++ {
		_, err := GenerateIdentity(numBits)
		if err != nil {
			b.Fatal(err)
		}
	}
}
