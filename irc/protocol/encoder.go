package protocol

import (
	"bytes"
	"io"
)

func NewEncoder(writer io.Writer) Encoder {
	rv := &encoder{
		writer: writer,
	}
	return rv
}

type encoder struct {
	writer io.Writer
}

func (e *encoder) Encode(msg *Message) error {
	buf := &bytes.Buffer{}
	buf.WriteString(msg.Marshal())
	buf.WriteString("\r\n")
	buf.WriteTo(e.writer)
	return nil
}
