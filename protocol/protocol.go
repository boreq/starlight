package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/boreq/lainnet/protocol/message"
	"github.com/boreq/lainnet/utils"
	"github.com/golang/protobuf/proto"
)

var log = utils.GetLogger("protocol")

func Encode(msg proto.Message) ([]byte, error) {
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

func Decode(data []byte) (proto.Message, error) {
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
	case 9:
		msg = &message.PrivateMessage{}
	case 10:
		msg = &message.ChannelMessage{}
	case 11:
		msg = &message.StorePubKey{}
	case 12:
		msg = &message.FindPubKey{}
	case 13:
		msg = &message.StoreChannel{}
	case 14:
		msg = &message.FindChannel{}
	default:
		log.Debugf("Decode: unknown message type %d", cmd)
		return nil, errors.New("Unknown message type")
	}
	err := proto.Unmarshal(buf.Bytes(), msg)
	return msg, err
}
