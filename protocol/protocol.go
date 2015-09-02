// This package implements the protocol used to exchange information between
// nodes participating in the network.
//
// Top level protocol structure:
//     LEN      TYPE      DESCRIPTION
//     4        uint32    Size of the message payload.
//     ?        []byte    Message payload.
//
// Protocol operates in two states. Handshake is performed in 'basic' mode and
// after completing it a switch to a 'secure' mode is made.
//
// Basic mode payload structure:
//     LEN      TYPE      DESCRIPTION
//     4        uint32    Type of the message.
//     size-4   []byte    Protobuf encoded message.
//
// Secure mode payload structure:
//     LEN      TYPE      DESCRIPTION
//     ?        []byte    HMAC, length depends on hash type.
//     size-?   []byte    Encrypted basic mode payload.
package protocol

import (
	"bytes"
	"encoding/binary"
)

// Unmarshaler collects the data written to it and decodes it as defined by
// the top level protocol structure.
type Unmarshaler interface {
	// Write adds more data to be decoded.
	Write([]byte) (int, error)
}

// Marshal encodes the data as defined by the top level protocol structure.
func Marshal(payload []byte) ([]byte, error) {
	buf := &bytes.Buffer{}
	if err := binary.Write(buf, binary.BigEndian, uint32(len(payload))); err != nil {
		return nil, err
	}
	buf.Write(payload)
	return buf.Bytes(), nil
}
