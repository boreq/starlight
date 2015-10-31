package basic

import (
	"bytes"
	"testing"
)

func TestBasic(t *testing.T) {
	data := []byte("data")

	buf := &bytes.Buffer{}
	e := NewEncoder(buf)
	err := e.Encode(data)
	if err != nil {
		t.Fatal(err)
	}

	d := NewDecoder(buf)
	decodedData, err := d.Decode()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(data, decodedData) {
		t.Fatal("Decoded data is different")
	}
}
