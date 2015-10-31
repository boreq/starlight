// Package basic implements a simple decoder and encoder.
//
// Structure of the sent data:
//     LEN      TYPE      DESCRIPTION
//     4        uint32    Size of the payload.
//     ?        []byte    Payload.
package basic

import (
	"github.com/boreq/lainnet/transport"
	"io"
)

func New(rw io.ReadWriter) (transport.Encoder, transport.Decoder) {
	return NewEncoder(rw), NewDecoder(rw)
}
