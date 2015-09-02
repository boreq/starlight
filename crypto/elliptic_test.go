package crypto

import (
	"bytes"
	"testing"
)

func TestSharedSecret(t *testing.T) {
	var curveName = "P224"

	curve, err := GetCurve(curveName)
	if err != nil {
		t.Fatal(err)
	}

	// Local
	local, err := GenerateEphemeralKeypair(curve)
	if err != nil {
		t.Fatal(err)
	}

	localBytes, err := local.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	// Remote
	remote, err := GenerateEphemeralKeypair(curve)
	if err != nil {
		t.Fatal(err)
	}

	remoteBytes, err := remote.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	// Secrets
	sharedSecret1, err := local.GenerateSharedSecret(remoteBytes)
	if err != nil {
		t.Fatal(err)
	}

	sharedSecret2, err := remote.GenerateSharedSecret(localBytes)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(sharedSecret1, sharedSecret2) {
		t.Fatal("Shared secrets are different")
	}
}
