package secure

import (
	"bytes"
	"encoding/binary"
	"errors"
)

const paddingSizeLen = 1

func addPadding(data []byte, blockSize int) ([]byte, error) {
	buf := &bytes.Buffer{}

	totalLen := paddingSizeLen + len(data)
	paddingLen := uint8(blockSize - (totalLen % blockSize))

	// Write padding length
	if err := binary.Write(buf, binary.BigEndian, paddingLen); err != nil {
		return nil, err
	}

	// Write data
	buf.Write(data)

	// Write padding
	buf.Write(make([]byte, paddingLen))

	return buf.Bytes(), nil
}

func stripPadding(data []byte) ([]byte, error) {
	buf := &bytes.Buffer{}
	buf.Write(data)

	// Read padding length
	var paddingSize uint8
	if err := binary.Read(buf, binary.BigEndian, &paddingSize); err != nil {
		return nil, err
	}

	// Sanity
	if int(paddingSize) > buf.Len() {
		return nil, errors.New("Padding size bigger than the data length")
	}

	// Strip padding
	rw := make([]byte, buf.Len()-int(paddingSize))
	buf.Read(rw)
	return rw, nil
}
