package basic

import (
	"bytes"
	"testing"
)

// TestBasic checks if the data is correctly encoded and then decoded.
func TestBasic(t *testing.T) {
	data := []byte("data")

	in := bytes.NewBuffer(data)
	out := &bytes.Buffer{}
	b := New()

	// Encode
	err := b.Encode(in, out)
	if err != nil {
		t.Fatal(err)
	}

	// Decode
	in.Reset()
	err = b.Decode(out, in)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(data, in.Bytes()) {
		t.Fatal("Decoded data is different")
	}
}
