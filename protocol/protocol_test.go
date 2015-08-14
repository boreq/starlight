package protocol

import (
	"errors"
	"testing"
	"time"
)

func TestMarshalUnmarshal(t *testing.T) {
	data := []byte("testing")
	b, err := Marshal(data)
	if err != nil {
		t.Fatal(err)
	}

	c := make(chan []byte)
	e := make(chan error)
	u := NewUnmarshaler(c)
	u.Write(b)
	go func() {
		select {
		case payload, ok := <-c:
			if !ok {
				e <- errors.New("Channel closed")
			} else {
				if len(payload) != len(data) {
					t.Fatal("Invalid length received")
				}
			}
			close(e)
		case <-time.After(1 * time.Second):
			e <- errors.New("Did not return a message")
		}
	}()

	err = <-e
	if err != nil {
		t.Fatal(err)
	}
}
