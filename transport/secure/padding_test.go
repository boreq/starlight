package secure

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

	buf := bytes.NewBuffer(data)
	err = stripPadding(buf)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(buf.Bytes(), original) {
		t.FailNow()
	}
}
