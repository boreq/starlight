package protocol

import (
	"bufio"
	"io"
	"strings"
)

func NewDecoder(reader io.Reader) Decoder {
	rv := &decoder{
		reader: bufio.NewReader(reader),
	}
	return rv
}

type decoder struct {
	reader *bufio.Reader
}

func (d *decoder) Decode() (*Message, error) {
	line, err := d.reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSpace(line)
	msg := UnmarshalMessage(line)
	return msg, nil
}
