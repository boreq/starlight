package protocol

import (
	"bytes"
	"golang.org/x/net/context"
	"testing"
	"time"
)

func TestMarshalUnmarshal(t *testing.T) {
	// Marshal
	data := []byte("testing")
	b, err := Marshal(data)
	if err != nil {
		t.Fatal(err)
	}

	// Unmarshal
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	c := make(chan []byte)
	u := NewUnmarshaler(ctx, c)
	go func() {
		if _, err := u.Write(b); err != nil {
			t.Fatal(err)
		}
	}()
	select {
	case payload, ok := <-c:
		if !ok {
			t.Fatal("Channel closed")
		} else if !bytes.Equal(payload, data) {
			t.Fatal("Invalid data received")
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}
