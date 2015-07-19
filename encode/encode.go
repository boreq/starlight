package encode

import "encoding/base64"

func Base64Encode(data []byte) []byte {
	l := base64.StdEncoding.EncodedLen(len(data))
	rw := make([]byte, l)
	base64.StdEncoding.Encode(rw, data)
	return rw
}

func Base64Decode(data []byte) []byte {
	rw := make([]byte, 0)
	base64.StdEncoding.Decode(rw, data)
	return rw
}
