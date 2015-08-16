package crypto

import (
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"math/big"
)

// GenerateEphemeralKey creates a new ephemeral keypair.
func GenerateEphemeralKeypair(curveName string) (EphemeralKey, error) {
	curve, err := getCurve(curveName)
	if err != nil {
		return nil, err
	}

	priv, x, y, err := elliptic.GenerateKey(curve, rand.Reader)
	rw := &ephemeralKey{
		curve: curve,
		priv:  priv,
		x:     x,
		y:     y,
	}
	return rw, err
}

// getCurve returns a curve based on the name.
func getCurve(name string) (elliptic.Curve, error) {
	switch name {
	case "P224":
		return elliptic.P224(), nil
	default:
		return nil, errors.New("Invalid curve name")
	}
}

type ephemeralKey struct {
	curve elliptic.Curve
	x, y  *big.Int
	priv  []byte
}

func (key *ephemeralKey) Bytes() ([]byte, error) {
	return elliptic.Marshal(key.curve, key.x, key.y), nil
}

func (key *ephemeralKey) GenerateSharedSecret(pub []byte) ([]byte, error) {
	x, y := elliptic.Unmarshal(key.curve, pub)
	// Bug in Go < 1.5, the point is not validated. Fixed in
	// d86b8d34d069c3895721ba47cac664f8bbf2b8ad
	if x == nil || y == nil || !key.curve.IsOnCurve(x, y) {
		return nil, errors.New("Invalid public key")
	}
	secretX, _ := key.curve.ScalarMult(x, y, key.priv)
	return secretX.Bytes(), nil
}
