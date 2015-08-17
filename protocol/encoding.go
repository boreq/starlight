package protocol

import (
	"bytes"
	"encoding/binary"
	"golang.org/x/net/context"
)

func NewUnmarshaler(ctx context.Context, c chan<- []byte) Unmarshaler {
	u := &unmarshaler{
		ctx: ctx,
		c:   c,
	}
	return u
}

type unmarshaler struct {
	ctx context.Context
	c   chan<- []byte
	buf bytes.Buffer
}

func (u *unmarshaler) Write(d []byte) (int, error) {
	n, _ := u.buf.Write(d)
	err := u.process()
	return n, err
}

const msgHeaderSize = 4

func (u *unmarshaler) process() error {
	for {
		// Do we have enough data to read header?
		if u.buf.Len() < msgHeaderSize {
			return nil
		}

		// Read header.
		size, err := readMsgSize(u.buf.Bytes()[:msgHeaderSize])
		if err != nil {
			return err
		}

		// Do we have enough data to read entire message?
		totalSize := size + msgHeaderSize
		if uint32(u.buf.Len()) < totalSize {
			return nil
		}

		// Decode.
		trash := make([]byte, msgHeaderSize)
		u.buf.Read(trash)
		payload := make([]byte, size)
		u.buf.Read(payload)

		select {
		case u.c <- payload:
		case <-u.ctx.Done():
		}
		return nil
	}
}

// readMsgSize reads the message size from b.
func readMsgSize(b []byte) (size uint32, err error) {
	buf := bytes.NewBuffer(b)
	err = binary.Read(buf, binary.BigEndian, &size)
	return
}
