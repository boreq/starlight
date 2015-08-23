package encoder

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/boreq/netblog/protocol/message"
	"github.com/golang/protobuf/proto"
)

// Basic encoder encodes protobuf messages in the basic protocol mode.
func NewBasic() Encoder {
	return &basic{}
}

type basic struct{}

func (b *basic) Encode(msg proto.Message) ([]byte, error) {
	buf := &bytes.Buffer{}

	// Command.
	cmd, err := cmdEncode(msg)
	if err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, cmd); err != nil {
		return nil, err
	}

	// Payload.
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	buf.Write(data)

	return buf.Bytes(), nil
}

func (b *basic) Decode(data []byte) (proto.Message, error) {
	buf := bytes.NewBuffer(data)

	// Decode command type.
	var cmd uint32
	if err := binary.Read(buf, binary.BigEndian, &cmd); err != nil {
		return nil, err
	}

	// Payload. Unfortunately the switch has to be hardcoded.
	var msg proto.Message
	switch cmd {
	case 1:
		msg = &message.Init{}
	case 2:
		msg = &message.Handshake{}
	case 3:
		msg = &message.ConfirmHandshake{}
	case 4:
		msg = &message.Identity{}
	case 5:
		msg = &message.Ping{}
	case 6:
		msg = &message.Pong{}
	case 7:
		msg = &message.FindNode{}
	case 8:
		msg = &message.Nodes{}
	default:
		return nil, errors.New("Unknown message type")
	}
	err := proto.Unmarshal(buf.Bytes(), msg)
	return msg, err
}
