package encode

import "encoding/base64"

func Base64Encode(data []byte) []byte {
	l := base64.StdEncoding.EncodedLen(len(data))
	rw := make([]byte, l)
	base64.StdEncoding.Encode(rw, data)
	return rw
}

func Base64Decode(data []byte) ([]byte, error) {
	rw := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
	n, err := base64.StdEncoding.Decode(rw, data)
	if err != nil {
		return nil, err
	}
	return rw[:n], nil
}
