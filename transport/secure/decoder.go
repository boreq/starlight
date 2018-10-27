package secure

import (
	"bytes"
	"crypto/cipher"
	"crypto/hmac"
	"encoding/binary"
	"errors"
	"github.com/boreq/starlight/crypto"
	"hash"
	"io"
)

func newDecoder(hmac hash.Hash, cipher cipher.BlockMode, nonce uint32) *decoder {
	rv := &decoder{
		hmac:   hmac,
		cipher: cipher,
		nonce:  nonce,
	}
	return rv
}

type decoder struct {
	hmac   hash.Hash
	cipher cipher.BlockMode
	nonce  uint32
}

func (d *decoder) decode(r io.Reader, w io.Writer) error {
	data := &bytes.Buffer{}
	n, err := data.ReadFrom(r)
	if err != nil {
		return err
	}
	log.Debugf("received %d bytes", n)

	// Check HMAC
	if data.Len() < d.hmac.Size() {
		return errors.New("HMAC missing")
	}
	receivedHm := make([]byte, d.hmac.Size())
	_, err = data.Read(receivedHm)
	if err != nil {
		return err
	}
	expectedHm := crypto.Digest(d.hmac, data.Bytes())
	if !hmac.Equal(receivedHm, expectedHm) {
		return errors.New("Invalid HMAC")
	}

	// Decrypt payload
	if data.Len()%d.cipher.BlockSize() != 0 {
		return errors.New("Invalid length of the encrypted data")
	}
	bufContent := data.Bytes()
	d.cipher.CryptBlocks(bufContent, bufContent)

	// Strip padding
	err = stripPadding(data)
	if err != nil {
		return err
	}

	// Check nonce
	var nonce uint32
	if err := binary.Read(data, binary.BigEndian, &nonce); err != nil {
		return err
	}
	if nonce != d.nonce {
		return errors.New("Invalid nonce")
	}
	d.nonce++

	// Write the data
	_, err = data.WriteTo(w)
	if err != nil {
		return err
	}
	return nil
}
