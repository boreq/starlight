package core

import (
	"fmt"
	"github.com/boreq/lainnet/protocol/message"
	"testing"
)

func BenchmarkChannelMessageCryptoPuzzle(b *testing.B) {
	var nonce uint64
	var timestamp int64 = 10
	text := "some kind of a message"
	msg := &message.ChannelMessage{
		ChannelId: []byte("channel id"),
		NodeId:    []byte("node id"),
		Timestamp: &timestamp,
		Text:      &text,
		Nonce:     &nonce,
	}

	data, err := channelMessageDataToSign(msg)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		err := solveCryptoPuzzle(&nonce, data, 20)
		if err != nil {
			b.Fatal(err)
		}
	}

	fmt.Printf("\nnonce=%d\n", nonce)
}

func TestValidate(t *testing.T) {
	var nonce uint64
	var timestamp int64 = 10
	var text string = "some kind of a message"
	msg := &message.ChannelMessage{
		ChannelId: []byte("channel id"),
		NodeId:    []byte("node id"),
		Timestamp: &timestamp,
		Text:      &text,
		Nonce:     &nonce,
	}

	data, err := channelMessageDataToSign(msg)
	if err != nil {
		t.Fatal(err)
	}

	err = solveCryptoPuzzle(&nonce, data, 5)
	if err != nil {
		t.Fatal(err)
	}

	err = validateCryptoPuzzle(nonce, data, 5)
	if err != nil {
		t.Fatal(err)
	}

	err = validateCryptoPuzzle(1, data, 5)
	if err == nil {
		t.Fatal("Did not fail for the wrong nonce")
	}
}
