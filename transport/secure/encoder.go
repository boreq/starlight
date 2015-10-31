package secure

import (
	"bytes"
	"crypto/cipher"
	"github.com/boreq/lainnet/crypto"
	"github.com/boreq/lainnet/transport"
	"github.com/boreq/lainnet/transport/basic"
	"hash"
	"io"
)

func NewEncoder(writer io.Writer, hmac hash.Hash, cipher cipher.BlockMode) transport.Encoder {
	rv := &encoder{
		writer: writer,
		hmac:   hmac,
		cipher: cipher,
	}
	return rv
}

type encoder struct {
	writer io.Writer
	hmac   hash.Hash
	cipher cipher.BlockMode
}

func (e *encoder) Encode(data []byte) error {
	// Encrypt payload
	data, err := addPadding(data, e.cipher.BlockSize())
	if err != nil {
		return err
	}
	e.cipher.CryptBlocks(data, data)

	// Calculate HMAC
	hm := crypto.Digest(e.hmac, data)

	// Concat HMAC and data
	buf := &bytes.Buffer{}
	buf.Write(hm)
	buf.Write(data)

	// Encapsulate using the basic encoder
	enc := basic.NewEncoder(e.writer)
	return enc.Encode(buf.Bytes())
}
