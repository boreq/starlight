package encoder

import (
	"bytes"
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
