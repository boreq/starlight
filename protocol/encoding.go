package protocol

import (
	"bytes"
	"encoding/binary"
	"github.com/boreq/netblog/utils"
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
	u.process()
	return n, nil
}

const msgHeaderSize = 4

var umLog = utils.Logger("Unmarshaller")

func (u *unmarshaler) process() {
	for {
		// Do we have enough data to read header?
		if u.buf.Len() < msgHeaderSize {
			return
		}

		// Read header.
		size, err := readMsgSize(u.buf.Bytes()[:msgHeaderSize])
		if err != nil {
			// TODO: close chanel and handle protocol error
			umLog.Print("Failed to read message header, panic")
			panic(err)
		}

		// Do we have enough data to read entire message?
		totalSize := size + msgHeaderSize
		if uint32(u.buf.Len()) < totalSize {
			return
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
	}
}

// readMsgSize reads the message size from b.
func readMsgSize(b []byte) (size uint32, err error) {
	buf := bytes.NewBuffer(b)
	err = binary.Read(buf, binary.BigEndian, &size)
	return
}
