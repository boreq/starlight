package encoder

import (
	"github.com/boreq/netblog/protocol/message"
	"testing"
)

func TestEncode(t *testing.T) {
	msg := &message.Init{
		PubKey: []byte{},
	}
	e := NewBasic()

	b, err := e.Encode(msg)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Encoded len", len(b))

	_, err = e.Decode(b)
	if err != nil {
		t.Fatal(err)
	}
}
