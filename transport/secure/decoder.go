package secure

import (
	"crypto/cipher"
	"crypto/hmac"
	"errors"
	"github.com/boreq/lainnet/crypto"
	"github.com/boreq/lainnet/transport"
	"github.com/boreq/lainnet/transport/basic"
	"hash"
	"io"
)

func NewDecoder(reader io.Reader, hmac hash.Hash, cipher cipher.BlockMode) transport.Decoder {
	rv := &decoder{
		reader: reader,
		hmac:   hmac,
		cipher: cipher,
	}
	return rv
}

type decoder struct {
	reader io.Reader
	hmac   hash.Hash
	cipher cipher.BlockMode
}

const sizeHeaderLen = 4

func (d *decoder) Decode() ([]byte, error) {
	// Decapsulate using the basic decoder
	dec := basic.NewDecoder(d.reader)
	data, err := dec.Decode()
	if err != nil {
		return nil, err
	}

	// Check HMAC
	receivedHm := data[:d.hmac.Size()]
	data = data[d.hmac.Size():]
	expectedHm := crypto.Digest(d.hmac, data)
	if !hmac.Equal(receivedHm, expectedHm) {
		return nil, errors.New("Invalid HMAC")
	}

	// Decrypt payload
	d.cipher.CryptBlocks(data, data)
	return stripPadding(data)
}
