package basic

import (
	"bytes"
	"encoding/binary"
	"github.com/boreq/lainnet/transport"
	"io"
)

func NewDecoder(reader io.Reader) transport.Decoder {
	rv := &decoder{
		reader: reader,
	}
	return rv
}

type decoder struct {
	reader io.Reader
}

const sizeHeaderLen = 4

func (d *decoder) Decode() ([]byte, error) {
	// Get the size
	buf := make([]byte, sizeHeaderLen)
	_, err := io.ReadFull(d.reader, buf)
	if err != nil {
		return nil, err
	}

	// Decode the size
	size, err := readSizeHeader(buf)
	if err != nil {
		return nil, err
	}

	// Get the data
	rv := make([]byte, size)
	_, err = io.ReadFull(d.reader, rv)
	return rv, err
}

// readSizeHeader reads the message size from b.
func readSizeHeader(b []byte) (size uint32, err error) {
	buf := bytes.NewBuffer(b)
	err = binary.Read(buf, binary.BigEndian, &size)
	return
}
