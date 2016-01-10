package secure

import (
	"bytes"
	"crypto/cipher"
	"crypto/hmac"
	"encoding/binary"
	"errors"
	"github.com/boreq/lainnet/crypto"
	"github.com/boreq/lainnet/transport"
	"github.com/boreq/lainnet/transport/basic"
	"hash"
	"io"
)

func NewDecoder(reader io.Reader, hmac hash.Hash, cipher cipher.BlockMode, nonce uint32) transport.Decoder {
	rv := &decoder{
		reader: reader,
		hmac:   hmac,
		cipher: cipher,
		nonce:  nonce,
	}
	return rv
}

type decoder struct {
	reader io.Reader
	hmac   hash.Hash
	cipher cipher.BlockMode
	nonce  uint32
}

func (d *decoder) Decode() ([]byte, error) {
	// Decapsulate using the basic decoder
	dec := basic.NewDecoder(d.reader)
	data, err := dec.Decode()
	if err != nil {
		return nil, err
	}

	// Check HMAC
	if len(data) < d.hmac.Size() {
		return nil, errors.New("HMAC missing")
	}
	receivedHm := data[:d.hmac.Size()]
	data = data[d.hmac.Size():]
	expectedHm := crypto.Digest(d.hmac, data)
	if !hmac.Equal(receivedHm, expectedHm) {
		return nil, errors.New("Invalid HMAC")
	}

	// Decrypt payload
	if len(data)%d.cipher.BlockSize() != 0 {
		return nil, errors.New("Invalid length of the encrypted data")
	}
	d.cipher.CryptBlocks(data, data)

	// Strip padding
	data, err = stripPadding(data)
	if err != nil {
		return nil, err
	}

	// Check nonce
	var nonce uint32
	buf := bytes.NewBuffer(data)
	if err := binary.Read(buf, binary.BigEndian, &nonce); err != nil {
		return nil, err
	}

	//nonce := binary.BigEndian.Uint32(data[:8])
	if nonce != d.nonce {
		return nil, errors.New("Invalid nonce")
	}
	d.nonce++

	return buf.Bytes(), nil
}
