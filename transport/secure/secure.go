// Package secure implements a transport layer which provides encryption and
// integrity checks. Data is sent encrypted, with a nonce and an HMAC confirming
// its integrity added.
//
// Structure of the sent data:
//      ?        []byte    Payload HMAC, length depends on the hash type.
//      size-?   []byte    Encrypted nonce+payload, length: size - HMAC length.
package secure

import (
	"crypto/cipher"
	"github.com/boreq/lainnet/transport"
	"github.com/boreq/lainnet/utils"
	"hash"
	"io"
)

var log = utils.GetLogger("transport/secure")

func New(decoderHMAC, encoderHMAC hash.Hash, decoderCipher, encoderCipher cipher.BlockMode, decoderNonce, encoderNonce uint32) transport.Layer {
	enc := newEncoder(encoderHMAC, encoderCipher, encoderNonce)
	dec := newDecoder(decoderHMAC, decoderCipher, decoderNonce)
	rv := &secure{enc, dec}
	return rv
}

type secure struct {
	*encoder
	*decoder
}

func (s *secure) Encode(r io.Reader, w io.Writer) error {
	return s.encode(r, w)
}

func (s *secure) Decode(r io.Reader, w io.Writer) error {
	return s.decode(r, w)
}
