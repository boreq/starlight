package utils

import "testing"

func TestZerosLen(t *testing.T) {
	var a []byte

	a = []byte{0, 0}
	if rw := ZerosLen(a); rw != 16 {
		t.Fatal("First call invalid", rw)
	}

	a = []byte{0, 255}
	if rw := ZerosLen(a); rw != 8 {
		t.Fatal("Second call invalid", rw)
	}

	a = []byte{0, 37}
	if rw := ZerosLen(a); rw != 10 {
		t.Fatal("Third call invalid", rw)
	}

	a = []byte{}
	if rw := ZerosLen(a); rw != 0 {
		t.Fatal("Fourth call invalid", rw)
	}
}

func TestCompare(t *testing.T) {
	var a, b []byte

	a = []byte{0, 0}
	b = []byte{0, 0}
	if rw, err := Compare(a, b); rw != 0 || err != nil {
		t.Fatal("First call invalid", rw, err)
	}

	a = []byte{0, 0}
	b = []byte{0, 10}
	if rw, err := Compare(a, b); rw != -1 || err != nil {
		t.Fatal("Second call invalid", rw, err)
	}

	a = []byte{0, 37}
	b = []byte{0, 33}
	if rw, err := Compare(a, b); rw != 1 || err != nil {
		t.Fatal("Third call invalid", rw, err)
	}

	a = []byte{0, 0}
	b = []byte{0}
	if rw, err := Compare(a, b); err == nil {
		t.Fatal("Fourth call invalid", rw, err)
	}
}
