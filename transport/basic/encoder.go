package basic

import (
	"bytes"
	"encoding/binary"
	"github.com/boreq/lainnet/transport"
	"io"
)

func NewEncoder(writer io.Writer) transport.Encoder {
	rv := &encoder{
		writer: writer,
	}
	return rv
}

type encoder struct {
	writer io.Writer
}

func (e *encoder) Encode(data []byte) error {
	buf := &bytes.Buffer{}
	if err := binary.Write(buf, binary.BigEndian, uint32(len(data))); err != nil {
		return err
	}
	buf.Write(data)
	_, err := buf.WriteTo(e.writer)
	return err
}
