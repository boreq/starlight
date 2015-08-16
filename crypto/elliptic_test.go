package crypto

import (
	"github.com/boreq/netblog/utils"
	"testing"
)

func TestSharedSecret(t *testing.T) {
	var curveName = "P224"

	// Local
	local, err := GenerateEphemeralKeypair(curveName)
	if err != nil {
		t.Fatal(err)
	}

	localBytes, err := local.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	// Remote
	remote, err := GenerateEphemeralKeypair(curveName)
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

	result, err := utils.Compare(sharedSecret1, sharedSecret2)
	if err != nil || result != 0 {
		t.Fatal("Shared secrets are different")
	}
}
