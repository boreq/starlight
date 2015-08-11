package protocol

import (
	"github.com/boreq/netblog/protocol/message"
	"testing"
)

func TestMarshal(t *testing.T) {
	m := &message.Init{}
	b, err := Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("len", len(b))
}

func TestDual(t *testing.T) {
	m := &message.Init{}
	b, err := Marshal(m)
	if err != nil {
		t.Fatal(err)
	}

	u := NewUnmarshaler()
	u.Write(b)
}
