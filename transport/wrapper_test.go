package transport

import (
	"bytes"
	"io"
	"testing"
)

type mock struct{}

func (m *mock) Encode(r io.Reader, w io.Writer) error {
	_, err := io.Copy(w, r)
	return err
}

func (m *mock) Decode(r io.Reader, w io.Writer) error {
	_, err := io.Copy(w, r)
	return err
}

func TestErrorsWithNoLayers(t *testing.T) {
	r := &bytes.Buffer{}
	w := &bytes.Buffer{}
	wrapper := NewWrapper(r, w)

	err := wrapper.Send([]byte("test"))
	if err == nil {
		t.Fatal("Should fail with no layers")
	}

	_, err = wrapper.Receive()
	if err == nil {
		t.Fatal("Should fail with no layers")
	}
}

func TestNoPanics(t *testing.T) {
	r := &bytes.Buffer{}
	w := &bytes.Buffer{}
	wrapper := NewWrapper(r, w)
	wrapper.AddLayer(&mock{})

	// Send
	err := wrapper.Send([]byte("testSend"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal([]byte("testSend"), w.Bytes()) {
		t.FailNow()
	}

	// Receive
	r.WriteString("testReceive")
	data, err := wrapper.Receive()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal([]byte("testReceive"), data) {
		t.FailNow()
	}
}
