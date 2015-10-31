package protocol

import (
	"github.com/boreq/lainnet/protocol/message"
	"testing"
)

func TestEncode(t *testing.T) {
	var pingRandom uint32 = 0

	msg := &message.Ping{
		Random: &pingRandom,
	}

	// Encode.
	b, err := Encode(msg)
	if err != nil {
		t.Fatal(err)
	}

	// Decode.
	dMsg, err := Decode(b)
	if err != nil {
		t.Fatal(err)
	}
	if pMsg, ok := dMsg.(*message.Ping); !ok || *pMsg.Random != pingRandom {
		t.Fatal("Invalid message decoded")
	}
}
