package protocol

import (
	"github.com/golang/protobuf/proto"
)

type MessageCommand int

const (
	Invalid MessageCommand = iota
	Init
)

// Message stores a command and a protobuf encoded payload.
type Message struct {
	Command MessageCommand
	Payload []byte
}

// Unmarshaler collects the data written to it and assembles it into the
// complete message structs which are then sent through a channel.
type Unmarshaler interface {
	// Write adds more data to be decoded.
	Write([]byte) (int, error)
}

func Marshal(msgCommand MessageCommand, msg proto.Message) (*Message, error) {
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	rw := &Message{
		Command: msgCommand,
		Payload: data,
	}
	return rw, nil
}
