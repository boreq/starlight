package crypto

import (
	"strings"
	"testing"
)

func TestGetCurve(t *testing.T) {
	data := strings.Split(SupportedCurves, ",")
	for _, name := range data {
		_, err := getCurve(name)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestGetHash(t *testing.T) {
	data := strings.Split(SupportedHashes, ",")
	for _, name := range data {
		_, err := GetHash(name)
		if err != nil {
			t.Fatal(err)
		}
	}
}
