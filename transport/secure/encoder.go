package secure

import (
	"bytes"
	"crypto/cipher"
	"encoding/binary"
	"github.com/boreq/starlight/crypto"
	"hash"
	"io"
)

func newEncoder(hmac hash.Hash, cipher cipher.BlockMode, nonce uint32) *encoder {
	rv := &encoder{
		hmac:   hmac,
		cipher: cipher,
		nonce:  nonce,
	}
	return rv
}

type encoder struct {
	hmac   hash.Hash
	cipher cipher.BlockMode
	nonce  uint32
}

func (e *encoder) encode(r io.Reader, w io.Writer) error {
	buf := &bytes.Buffer{}

	// Put the nonce and the payload in the buffer
	if err := binary.Write(buf, binary.BigEndian, e.nonce); err != nil {
		return err
	}
	_, err := buf.ReadFrom(r)
	if err != nil {
		return err
	}
	e.nonce++

	// Add padding to the contents of the buffer and encrypt them
	data, err := addPadding(buf.Bytes(), e.cipher.BlockSize())
	if err != nil {
		return err
	}
	e.cipher.CryptBlocks(data, data)

	// Calculate the HMAC
	hm := crypto.Digest(e.hmac, data)

	// Concat the HMAC and data and write the result to the writer
	buf.Reset()
	buf.Write(hm)
	buf.Write(data)
	n, err := buf.WriteTo(w)
	if err != nil {
		return err
	}
	log.Debugf("written %d bytes", n)
	return nil
}
