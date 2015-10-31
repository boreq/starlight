// Package secure implements a secure decoder and encoder. Data is sent
// encrypted and with a HMAC confirming its integrity.
//
// Structure of the sent data:
//     LEN      TYPE      DESCRIPTION
//     4        uint32    Size of the payload.
//     ?        []byte    Payload HMAC, length depends on the hash type.
//     size-?   []byte    Encrypted payload, length: size - HMAC length.
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
