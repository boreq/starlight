// Package basic implements a simple transport layer. It adds information about
// the size of the payload to the data so that it is possible to send variable
// length data packets via a stream connection.
//
// Structure of the sent data:
//     LEN      TYPE      DESCRIPTION
//     4        uint32    Size of the payload received from the reader.
//     ?        []byte    Payload received from the reader.
package basic

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/boreq/starlight/transport"
	"github.com/boreq/starlight/utils"
	"io"
	"math"
)

var log = utils.GetLogger("transport/basic")

// MaxMessageLen defines the max length of a message which can be sent or
// received using this layer.
const MaxMessageLen = math.MaxUint32

func New(messageSizeLimit uint32) transport.Layer {
	return &basic{
		MessageSizeLimit: messageSizeLimit,
	}
}

type basic struct {
	// MessageSizeLimit limits the size of sent and received messages. The size
	// is represented in bytes.
	MessageSizeLimit uint32
}

func (b *basic) Encode(r io.Reader, w io.Writer) error {
	buf := &bytes.Buffer{}
	_, err := buf.ReadFrom(r)
	if err != nil {
		return err
	}

	// Write the size
	if buf.Len() > MaxMessageLen {
		return errors.New("data length exceeding MaxMessageLen")
	}
	if uint32(buf.Len()) > b.MessageSizeLimit {
		return errors.New("data length exceeding MessageSizeLimit")
	}
	if err := binary.Write(w, binary.BigEndian, uint32(buf.Len())); err != nil {
		return err
	}

	// Write the data
	n, err := io.Copy(w, buf)
	if err != nil {
		return err
	}
	log.Debugf("written %d bytes", n+sizeHeaderLen)
	return nil
}

const sizeHeaderLen = 4

func (b *basic) Decode(r io.Reader, w io.Writer) error {
	// Get the size
	buf := make([]byte, sizeHeaderLen)
	_, err := io.ReadFull(r, buf)
	if err != nil {
		return err
	}

	// Decode the size
	size, err := readSizeHeader(buf)
	if err != nil {
		return err
	}
	if size > MaxMessageLen {
		return errors.New("data length exceeding MaxMessageLen")
	}
	if size > b.MessageSizeLimit {
		return errors.New("data length exceeding MessageSizeLimit")
	}

	// Get the data
	_, err = io.CopyN(w, r, int64(size))
	log.Debugf("received %d bytes", size+sizeHeaderLen)
	return err
}

// readSizeHeader reads the message size from b.
func readSizeHeader(b []byte) (size uint32, err error) {
	buf := bytes.NewBuffer(b)
	err = binary.Read(buf, binary.BigEndian, &size)
	return
}
