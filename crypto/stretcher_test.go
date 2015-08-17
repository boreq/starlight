package crypto

import (
	"testing"
)

func TestInvalidCipher(t *testing.T) {
	_, _, err := StretchKey([]byte("secret"), []byte("salt"), "SHA512", "INVALID")
	if err == nil {
		t.Fatal("Did not fail")
	}
	t.Log("Correctly returned", err)
}

func TestInvalidHash(t *testing.T) {
	_, _, err := StretchKey([]byte("secret"), []byte("salt"), "INVALID", "AES-256")
	if err == nil {
		t.Fatal("Did not fail")
	}
	t.Log("Correctly returned", err)
}

func Test(t *testing.T) {
	_, _, err := StretchKey([]byte("secret"), []byte("salt"), "SHA256", "AES-256")
	if err != nil {
		t.Fatal(err)
	}
}
