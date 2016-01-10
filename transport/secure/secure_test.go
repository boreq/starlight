package secure

import (
	"bytes"
	"crypto"
	"crypto/cipher"
	"crypto/hmac"
	lcrypto "github.com/boreq/lainnet/crypto"
	"github.com/boreq/lainnet/transport/basic"
	"hash"
	"testing"
)

func get(t *testing.T) (encCipher, decCipher cipher.BlockMode, encHmac, decHmac hash.Hash) {
	const hash = crypto.SHA256
	const hashName = "SHA256"
	const cipherName = "AES-256"

	k1, _, err := lcrypto.StretchKey([]byte("a"), []byte("a"), hashName, cipherName)
	if err != nil {
		t.Fatal(err)
	}

	encBl, err := lcrypto.GetCipher(cipherName, k1.CipherKey)
	if err != nil {
		t.Fatal(err)
	}
	decBl, err := lcrypto.GetCipher(cipherName, k1.CipherKey)
	if err != nil {
		t.Fatal(err)
	}

	encHmac = hmac.New(hash.New, k1.MacKey)
	decHmac = hmac.New(hash.New, k1.MacKey)
	encCipher = cipher.NewCBCEncrypter(encBl, k1.IV)
	decCipher = cipher.NewCBCDecrypter(decBl, k1.IV)
	return
}

// TestSecure checks if the data is correctly encoded and then decoded.
func TestSecure(t *testing.T) {
	const nonce = 10
	data := []byte("data")
	encCipher, decCipher, encHmac, decHmac := get(t)

	buf := &bytes.Buffer{}
	e := NewEncoder(buf, encHmac, encCipher, nonce)
	err := e.Encode(data)
	if err != nil {
		t.Fatal(err)
	}

	d := NewDecoder(buf, decHmac, decCipher, nonce)
	decodedData, err := d.Decode()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(data, decodedData) {
		t.Fatal("Decoded data is different")
	}
}

// TestSecureNonce checks if the decoding process will fail if the nonce differs.
func TestSecureNonce(t *testing.T) {
	const nonceEnc = 10
	const nonceDec = 11
	data := []byte("data")
	encCipher, decCipher, encHmac, decHmac := get(t)

	buf := &bytes.Buffer{}
	e := NewEncoder(buf, encHmac, encCipher, nonceEnc)
	err := e.Encode(data)
	if err != nil {
		t.Fatal(err)
	}

	d := NewDecoder(buf, decHmac, decCipher, nonceDec)
	_, err = d.Decode()
	if err == nil {
		t.Fatal("No error")
	}
}

// TestDecodeEmpty checks what happens if the data decoded by the basic encoder
// is completely empty.
func TestDecodeEmpty(t *testing.T) {
	_, decCipher, _, decHmac := get(t)

	buf := &bytes.Buffer{}
	e := basic.NewEncoder(buf)
	err := e.Encode([]byte{})
	if err != nil {
		t.Fatal(err)
	}

	d := NewDecoder(buf, decHmac, decCipher, 10)
	_, err = d.Decode()
	if err == nil {
		t.Fatal(err)
	}
}

// TestDecodeEmptyEncrypted checks what happens if the encrypted data is an
// empty slice but the HMAC is correct.
func TestDecodeEmptyEncrypted(t *testing.T) {
	_, decCipher, encHmac, decHmac := get(t)

	// Prepate a payload in which the encoded data is acutally missing
	data := []byte{}
	hm := lcrypto.Digest(encHmac, data)

	buf := &bytes.Buffer{}
	buf.Write(hm)
	buf.Write(data)

	encBuf := &bytes.Buffer{}
	enc := basic.NewEncoder(encBuf)
	err := enc.Encode(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	// Try to decode that data
	d := NewDecoder(encBuf, decHmac, decCipher, 10)
	_, err = d.Decode()
	if err == nil {
		t.Fatal("No error")
	}
}

// TestDecodeInvalidEncrypted checks what happens if the encrypted data has
// an invalid length not corresponding to the cipher's blocks size.
func TestDecodeInvalidEncrypted(t *testing.T) {
	_, decCipher, encHmac, decHmac := get(t)

	// Prepate a payload in which the encoded data is acutally missing
	data := []byte("a")
	hm := lcrypto.Digest(encHmac, data)

	buf := &bytes.Buffer{}
	buf.Write(hm)
	buf.Write(data)

	encBuf := &bytes.Buffer{}
	enc := basic.NewEncoder(encBuf)
	err := enc.Encode(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	// Try to decode that data
	d := NewDecoder(encBuf, decHmac, decCipher, 10)
	_, err = d.Decode()
	if err == nil {
		t.Fatal("No error")
	}
}
