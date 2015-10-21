package encoder

import (
	"bytes"
	"crypto/cipher"
	"crypto/hmac"
	"encoding/binary"
	"errors"
	"github.com/boreq/lainnet/crypto"
	"github.com/golang/protobuf/proto"
	"hash"
)

// Secure encoder encodes protobuf messages in the secure protocol mode.
func NewSecure(localKeys, remoteKeys crypto.StretchedKeys, hashName string, cipherName string) (Encoder, error) {
	hash, err := crypto.GetCryptoHash(hashName)
	if err != nil {
		return nil, err
	}

	localCipher, err := crypto.GetCipher(cipherName, localKeys.CipherKey)
	if err != nil {
		return nil, err
	}

	remoteCipher, err := crypto.GetCipher(cipherName, remoteKeys.CipherKey)
	if err != nil {
		return nil, err
	}

	rw := &secure{
		basic:        NewBasic(),
		localHmac:    hmac.New(hash.New, localKeys.MacKey),
		remoteHmac:   hmac.New(hash.New, remoteKeys.MacKey),
		localCipher:  cipher.NewCBCEncrypter(localCipher, localKeys.IV),
		remoteCipher: cipher.NewCBCDecrypter(remoteCipher, remoteKeys.IV),
	}
	return rw, nil
}

type secure struct {
	basic        Encoder
	localHmac    hash.Hash
	remoteHmac   hash.Hash
	localCipher  cipher.BlockMode
	remoteCipher cipher.BlockMode
}

func (s *secure) Encode(msg proto.Message) ([]byte, error) {
	buf := &bytes.Buffer{}

	// Payload.
	data, err := s.basic.Encode(msg)
	if err != nil {
		return nil, err
	}

	// Encode payload.
	data, err = addPadding(data, s.localCipher.BlockSize())
	s.localCipher.CryptBlocks(data, data)

	// HMAC.
	hm := crypto.Digest(s.localHmac, data)
	buf.Write(hm)
	buf.Write(data)

	return buf.Bytes(), nil
}

func (s *secure) Decode(data []byte) (proto.Message, error) {
	buf := bytes.NewBuffer(data)

	// HMAC.
	receivedHm := make([]byte, s.remoteHmac.Size())
	buf.Read(receivedHm)
	expectedHm := crypto.Digest(s.remoteHmac, buf.Bytes())
	if !hmac.Equal(receivedHm, expectedHm) {
		return nil, errors.New("Invalid HMAC")
	}

	// Decode payload.
	s.remoteCipher.CryptBlocks(buf.Bytes(), buf.Bytes())
	data, err := stripPadding(buf.Bytes())
	if err != nil {
		return nil, err
	}

	// Payload.
	msg, err := s.basic.Decode(data)

	return msg, err
}

var paddingSizeSize = 1

func addPadding(data []byte, blockSize int) ([]byte, error) {
	buf := &bytes.Buffer{}

	total := paddingSizeSize + len(data)
	paddingSize := uint8(blockSize - (total % blockSize))

	// Padding length.
	if err := binary.Write(buf, binary.BigEndian, paddingSize); err != nil {
		return nil, err
	}

	// Data.
	buf.Write(data)

	// Padding.
	buf.Write(make([]byte, paddingSize))

	return buf.Bytes(), nil
}

func stripPadding(data []byte) ([]byte, error) {
	buf := &bytes.Buffer{}
	buf.Write(data)

	// Read padding length.
	var paddingSize uint8
	if err := binary.Read(buf, binary.BigEndian, &paddingSize); err != nil {
		return nil, err
	}

	// Strip padding.
	rw := make([]byte, buf.Len()-int(paddingSize))
	buf.Read(rw)
	return rw, nil
}
