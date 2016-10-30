package secure

import (
	"bytes"
	"crypto"
	"crypto/cipher"
	"crypto/hmac"
	lcrypto "github.com/boreq/lainnet/crypto"
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

	in := bytes.NewBuffer(data)
	out := &bytes.Buffer{}

	e := newEncoder(encHmac, encCipher, nonce)
	err := e.encode(in, out)
	if err != nil {
		t.Fatal(err)
	}

	in.Reset()

	d := newDecoder(decHmac, decCipher, nonce)
	err = d.decode(out, in)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(data, in.Bytes()) {
		t.Fatal("Decoded data is different")
	}
}

// TestSecureNonce checks if the decoding process will fail if the nonce differs.
func TestSecureNonce(t *testing.T) {
	const nonceEnc = 10
	const nonceDec = 11
	data := []byte("data")
	encCipher, decCipher, encHmac, decHmac := get(t)

	in := bytes.NewBuffer(data)
	out := &bytes.Buffer{}

	e := newEncoder(encHmac, encCipher, nonceEnc)
	err := e.encode(in, out)
	if err != nil {
		t.Fatal(err)
	}

	in.Reset()

	d := newDecoder(decHmac, decCipher, nonceDec)
	err = d.decode(out, in)
	if err == nil {
		t.Fatal("No error")
	}
}

// TestDecodeEmpty checks what happens if the decoded data is empty.
func TestDecodeEmpty(t *testing.T) {
	_, decCipher, _, decHmac := get(t)
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}

	d := newDecoder(decHmac, decCipher, 10)
	err := d.decode(in, out)
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

	in := &bytes.Buffer{}
	in.Write(hm)
	in.Write(data)
	out := &bytes.Buffer{}

	// Try to decode that data
	d := newDecoder(decHmac, decCipher, 10)
	err := d.decode(in, out)
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

	in := &bytes.Buffer{}
	in.Write(hm)
	in.Write(data)
	out := &bytes.Buffer{}

	// Try to decode that data
	d := newDecoder(decHmac, decCipher, 10)
	err := d.decode(in, out)
	if err == nil {
		t.Fatal("No error")
	}
}
