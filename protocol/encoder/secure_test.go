package encoder

import (
	"bytes"
	"github.com/boreq/lainnet/crypto"
	"github.com/boreq/lainnet/protocol/message"
	"testing"
)

func TestPadding(t *testing.T) {
	original := []byte("hello")

	data, err := addPadding(original, 16)
	if err != nil {
		t.Fatal(err)
	}

	data, err = stripPadding(data)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(data, original) {
		t.FailNow()
	}
}

func TestSecure(t *testing.T) {
	var pingRandom uint32 = 0
	var hashName, cipherName = "SHA256", "AES-256"

	msg := &message.Ping{
		Random: &pingRandom,
	}
	k1, k2, err := crypto.StretchKey([]byte("sec"), []byte("salt"), hashName, cipherName)
	if err != nil {
		t.Fatal(err)
	}

	eLocal, err := NewSecure(k1, k2, hashName, cipherName)
	if err != nil {
		t.Fatal(err)
	}

	eRemote, err := NewSecure(k2, k1, hashName, cipherName)
	if err != nil {
		t.Fatal(err)
	}

	// Encode.
	b, err := eRemote.Encode(msg)
	if err != nil {
		t.Fatal(err)
	}

	// Decode.
	dMsg, err := eRemote.Decode(b)
	if err == nil {
		t.Fatal("Should fail, most likely with invalid HMAC")
	}

	dMsg, err = eLocal.Decode(b)
	if err != nil {
		t.Fatal(err)
	}
	if pMsg, ok := dMsg.(*message.Ping); !ok || *pMsg.Random != pingRandom {
		t.Fatal("Invalid message decoded")
	}
}
