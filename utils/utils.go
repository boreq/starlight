package utils

import (
	"errors"
	"os"
)

// If the directory doesn't exist creates it. If it is impossible returns an
// error or panics if the second argument is true.
func EnsureDirExists(path string, shouldPanic bool) error {
	err := os.MkdirAll(path, 0700)
	if err != nil && !os.IsExist(err) {
		if shouldPanic {
			panic(err)
		}
		return err
	}
	return nil
}

// Returns number of zero bits in the slice starting from the left.
func ZerosLen(a []byte) int {
	for i := 0; i < len(a); i++ {
		for j := 0; j < 8; j++ {
			mask := byte(1) << byte(7-j)
			if a[i]&mask != 0 {
				return i*8 + j
			}
		}
	}
	return len(a) * 8
}

var SliceLengthErr = errors.New("Length of the slices differs")

func XOR(a, b []byte) ([]byte, error) {
	if len(a) != len(b) {
		return nil, SliceLengthErr
	}

	rw := make([]byte, len(a))
	for i := 0; i < len(a); i++ {
		rw[i] = a[i] ^ b[i]
	}
	return rw, nil
}

// Cmp compares a and b and returns:
//
//   -1 if a <  b
//    0 if a == b
//   +1 if a >  b
//
func Compare(a, b []byte) (int, error) {
	if len(a) != len(b) {
		return 0, SliceLengthErr
	}

	for i := 0; i < len(a); i++ {
		for j := 0; j < 8; j++ {
			mask := byte(1) << byte(7-j)
			bitA := a[i] & mask
			bitB := b[i] & mask
			if bitA > bitB {
				return 1, nil
			}
			if bitA < bitB {
				return -1, nil
			}
		}
	}
	return 0, nil
}
