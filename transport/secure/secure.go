// Package secure implements a secure decoder and encoder. Data is sent
// encrypted and with a HMAC confirming its integrity. The secure encoder uses
// the basic encoder to encapsulate the data.
//
// Structure of the data sent in the payload field of the basic encoder:
//      ?        []byte    Payload HMAC, length depends on the hash type.
//      size-?   []byte    Encrypted payload, length: size - HMAC length.
package secure

import (
	"crypto/cipher"
	"github.com/boreq/lainnet/transport"
	"hash"
	"io"
)

func New(rw io.ReadWriter, decoderHMAC, encoderHMAC hash.Hash, decoderCipher, encoderCipher cipher.BlockMode) (transport.Encoder, transport.Decoder) {
	enc := NewEncoder(rw, encoderHMAC, encoderCipher)
	dec := NewDecoder(rw, decoderHMAC, decoderCipher)
	return enc, dec
}
