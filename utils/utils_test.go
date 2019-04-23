package utils

import (
	"bytes"
	"testing"
)

var zerosLenTests = []struct {
	a []byte
	i int
}{
	{[]byte{}, 0},
	{[]byte{0, 0}, 16},
	{[]byte{0, 37}, 10},
}

func TestZerosLen(t *testing.T) {
	for _, tt := range zerosLenTests {
		if i := ZerosLen(tt.a); i != tt.i {
			t.Fatalf(`ZerosLen(%x), want %v, got %v`, tt.a, tt.i, i)
		}
	}
}

func TestXORDifferentLength(t *testing.T) {
	a := []byte{0, 0}
	b := []byte{0}
	if _, err := XOR(a, b); err != SliceLengthErr {
		t.Fatalf("XOR(%x, %x) wrong error: %s", a, b, err)
	}
}

func TestXORIdenticalLength(t *testing.T) {
	a := []byte{0}
	b := []byte{0}
	if _, err := XOR(a, b); err != nil {
		t.Fatalf("XOR(%x, %x) failed despite the identical length: %s", a, b, err)
	}
}

var xorTests = []struct {
	a, b, c []byte
}{
	{[]byte{}, []byte{}, []byte{}},
	{[]byte{0, 0}, []byte{0, 0}, []byte{0, 0}},
	{[]byte{7, 7}, []byte{7, 7}, []byte{0, 0}},
	{[]byte{240, 15}, []byte{240, 240}, []byte{0, 255}},
	{[]byte{240, 15}, []byte{0, 0}, []byte{240, 15}},
}

func TestXOR(t *testing.T) {
	for _, tt := range xorTests {
		if c, err := XOR(tt.a, tt.b); !bytes.Equal(c, tt.c) || err != nil {
			t.Fatalf(`XOR(%x, %x), want %v, got %v`, tt.a, tt.b, tt.c, c)
		}
	}
}

func TestCompareDifferentLength(t *testing.T) {
	a := []byte{0, 0}
	b := []byte{0}
	if _, err := Compare(a, b); err != SliceLengthErr {
		t.Fatalf("Compare(%x, %x) wrong error: %s", a, b, err)
	}
}

func TestCompareIdenticalLength(t *testing.T) {
	a := []byte{0}
	b := []byte{0}
	if _, err := Compare(a, b); err != nil {
		t.Fatalf("Compare(%x, %x) failed despite the identical length: %s", a, b, err)
	}
}

var compareTests = []struct {
	a, b []byte
	i    int
}{
	{[]byte{}, []byte{}, 0},
	{[]byte{0, 0}, []byte{0, 0}, 0},
	{[]byte{0, 37}, []byte{0, 33}, 1},
	{[]byte{0, 33}, []byte{0, 37}, -1},
}

func TestCompare(t *testing.T) {
	for _, tt := range compareTests {
		if i, err := Compare(tt.a, tt.b); i != tt.i || err != nil {
			t.Fatalf(`Compare(%x, %x), want %v, got %v`, tt.a, tt.b, tt.i, i)
		}
	}
}
