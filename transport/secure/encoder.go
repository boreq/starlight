package secure

import (
	"bytes"
	"crypto/cipher"
	"encoding/binary"
	"github.com/boreq/lainnet/crypto"
	"github.com/boreq/lainnet/transport"
	"github.com/boreq/lainnet/transport/basic"
	"hash"
	"io"
)

func NewEncoder(writer io.Writer, hmac hash.Hash, cipher cipher.BlockMode, nonce uint32) transport.Encoder {
	rv := &encoder{
		writer: writer,
		hmac:   hmac,
		cipher: cipher,
		nonce:  nonce,
	}
	return rv
}

type encoder struct {
	writer io.Writer
	hmac   hash.Hash
	cipher cipher.BlockMode
	nonce  uint32
}

func (e *encoder) Encode(data []byte) error {
	buf := &bytes.Buffer{}

	// Add a nonce to the payload
	if err := binary.Write(buf, binary.BigEndian, e.nonce); err != nil {
		return err
	}
	buf.Write(data)
	e.nonce++

	// Encrypt payload
	data, err := addPadding(buf.Bytes(), e.cipher.BlockSize())
	if err != nil {
		return err
	}
	e.cipher.CryptBlocks(data, data)

	// Calculate HMAC
	hm := crypto.Digest(e.hmac, data)

	// Concat HMAC and data
	buf.Reset()
	buf.Write(hm)
	buf.Write(data)

	// Encapsulate using the basic encoder
	enc := basic.NewEncoder(e.writer)
	return enc.Encode(buf.Bytes())
}
