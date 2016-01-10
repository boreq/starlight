package basic

import (
	"bytes"
	"testing"
)

// TestBasic checks if data slices are correctly encoded and then decoded.
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

// TestDecode checks if there are no issues if a reader returns an error.
func TestDecode(t *testing.T) {
	buf := bytes.NewBuffer([]byte{})

	d := NewDecoder(buf)
	_, err := d.Decode()
	if err == nil {
		t.Fatal("No error")
	}
}
