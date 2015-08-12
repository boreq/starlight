package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/boreq/netblog/utils"
)

func Marshal(msg Message) ([]byte, error) {
	return encodeMessage(msg)
}

func NewUnmarshaler(c chan<- Message) Unmarshaler {
	u := &unmarshaler{
		c: c,
	}
	return u
}

// Encodes a message to bytes.
func encodeMessage(msg Message) ([]byte, error) {
	b := &bytes.Buffer{}
	cmd, err := encodeCommand(msg.Command)
	if err != nil {
		return nil, err
	}

	if err := binary.Write(b, binary.BigEndian, cmd); err != nil {
		return nil, err
	}
	if err := binary.Write(b, binary.BigEndian, uint32(len(msg.Payload))); err != nil {
		return nil, err
	}
	b.Write(msg.Payload)
	return b.Bytes(), nil
}

// Maps localy used MessageCommand type to the actual values used in the encoded
// messages.
var cmdMap = map[MessageCommand]uint32{
	Init: 1,
}

func encodeCommand(command MessageCommand) (uint32, error) {
	rw, ok := cmdMap[command]
	if !ok {
		return 0, errors.New("Unknown command")
	}
	return rw, nil
}

func decodeCommand(command uint32) (MessageCommand, error) {
	for key, value := range cmdMap {
		if value == command {
			return key, nil
		}
	}
	return Invalid, errors.New("Unknown command")
}

// Just a helper.
type msgHeader struct {
	Command uint32
	Size    uint32
}

// readMsgHeader reads the message header from b and returns
// (command, size, error) triple.
func readMsgHeader(b []byte) (*msgHeader, error) {
	buf := bytes.NewBuffer(b)
	rw := &msgHeader{}
	if err := binary.Read(buf, binary.BigEndian, &rw.Command); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.BigEndian, &rw.Size); err != nil {
		return nil, err
	}
	return rw, nil
}

type unmarshaler struct {
	c   chan<- Message
	buf bytes.Buffer
}

func (u *unmarshaler) Write(d []byte) (int, error) {
	n, _ := u.buf.Write(d)
	u.process()
	return n, nil
}

const msgHeaderSize = 8

var umLog = utils.Logger("Unmarshaller")

func (u *unmarshaler) process() {
	for {
		// Do we have enough data to read header?
		if u.buf.Len() < msgHeaderSize {
			return
		}

		// Read header.
		h, err := readMsgHeader(u.buf.Bytes()[:msgHeaderSize])
		if err != nil {
			umLog.Print("Failed to read message header, panic")
			panic(err)
		}

		// Do we have enough data to read entire message?
		totalSize := h.Size + msgHeaderSize
		if uint32(u.buf.Len()) < totalSize {
			return
		}

		// Decode.
		cmd, err := decodeCommand(h.Command)
		trash := make([]byte, 8)
		u.buf.Read(trash)
		payload := make([]byte, h.Size)
		u.buf.Read(payload)
		msg := Message{
			Command: cmd,
			Payload: payload,
		}
		// TODO
		go func() {
			u.c <- msg
		}()
	}
}
