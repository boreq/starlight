package protocol

import (
	"errors"
	"github.com/boreq/netblog/protocol/message"
	"testing"
)

func TestMarshal(t *testing.T) {
	m, err := NewMessage(Init, &message.Init{
		PubKey: make([]byte, 0),
	})
	if err != nil {
		t.Fatal(err)
	}

	b, err := Marshal(*m)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("len", len(b))
}

func TestDual(t *testing.T) {
	m, err := NewMessage(Init, &message.Init{
		PubKey: make([]byte, 0),
	})
	if err != nil {
		t.Fatal(err)
	}

	b, err := Marshal(*m)
	if err != nil {
		t.Fatal(err)
	}

	c := make(chan Message)
	e := make(chan error)
	u := NewUnmarshaler(c)
	u.Write(b)
	go func() {
		select {
		case msg, ok := <-c:
			if !ok {
				e <- errors.New("Channel closed")
			}
			t.Log(len(msg.Payload))
			close(e)
		default:
			e <- errors.New("Did not return a message")
		}
	}()

	err = <-e
	if err != nil {
		t.Fatal(err)
	}
}
