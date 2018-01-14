package peer

import (
	"bytes"
	"crypto/cipher"
	"crypto/hmac"
	"encoding/binary"
	"errors"
	"github.com/boreq/starlight/crypto"
	"github.com/boreq/starlight/network/node"
	"github.com/boreq/starlight/protocol/message"
	"github.com/boreq/starlight/transport"
	"github.com/boreq/starlight/transport/secure"
	"github.com/boreq/starlight/utils"
	"golang.org/x/net/context"
	"strings"
)

// selectParam accepts two strings containing words separated by commas. It
// returns the first word of the first string that is also present in the
// second string.
func selectParam(a, b string) string {
	sA := strings.Split(a, ",")
	sB := strings.Split(b, ",")
	for _, pA := range sA {
		for _, pB := range sB {
			if pA == pB {
				return pA
			}
		}
	}
	return ""
}

func newSecure(localKeys, remoteKeys crypto.StretchedKeys, localNonce, remoteNonce uint32, hashName string, cipherName string) (transport.Layer, error) {
	hash, err := crypto.GetCryptoHash(hashName)
	if err != nil {
		return nil, err
	}

	localCipher, err := crypto.GetCipher(cipherName, localKeys.CipherKey)
	if err != nil {
		return nil, err
	}

	remoteCipher, err := crypto.GetCipher(cipherName, remoteKeys.CipherKey)
	if err != nil {
		return nil, err
	}

	encHmac := hmac.New(hash.New, localKeys.MacKey)
	decHmac := hmac.New(hash.New, remoteKeys.MacKey)
	encCipher := cipher.NewCBCEncrypter(localCipher, localKeys.IV)
	decCipher := cipher.NewCBCDecrypter(remoteCipher, remoteKeys.IV)
	layer := secure.New(decHmac, encHmac, decCipher, encCipher, remoteNonce, localNonce)
	return layer, nil
}

// nonceSize is the size of the nonce used to calculate the shared secret. As
// the same nonce is converted to the uint32 nonce used by the secure encoder
// to prevent replay attacks, this value must be equal or higher than 32/8=4.
const nonceSize = 20

// The ride never ends. Performs a handshake, sets up a secure encoder and peer
// id.
func (p *peer) handshake(ctx context.Context, iden node.Identity) error {

	//
	// === EXCHANGE INIT MESSAGES ===
	//

	// Form an Init message.
	pubKeyBytes, err := iden.PubKey.Bytes()
	if err != nil {
		return err
	}

	localNonce := make([]byte, nonceSize)
	err = crypto.GenerateNonce(localNonce)
	if err != nil {
		return err
	}

	localInit := &message.Init{
		PubKey:           pubKeyBytes,
		Nonce:            localNonce,
		SupportedCurves:  &crypto.SupportedCurves,
		SupportedHashes:  &crypto.SupportedHashes,
		SupportedCiphers: &crypto.SupportedCiphers,
	}

	// Send Init message.
	err = p.SendWithContext(ctx, localInit)
	if err != nil {
		return err
	}

	// Receive Init message.
	msg, err := p.ReceiveWithContext(ctx)
	if err != nil {
		return err
	}
	remoteInit, ok := msg.(*message.Init)
	if !ok {
		return errors.New("The received message is not Init")
	}

	//
	// === PROCESS INIT MESSAGES ===
	//

	// Establish identity.
	remotePub, err := crypto.NewPublicKey(remoteInit.GetPubKey())
	if err != nil {
		return err
	}
	remoteId, err := remotePub.Hash()
	if err != nil {
		return err
	}

	// Fail if the id is invalid.
	if !node.ValidateId(remoteId) {
		return errors.New("Invalid remote id")
	}

	// Fail if the id is the same as the id of the local node.
	if node.CompareId(remoteId, iden.Id) {
		return errors.New("Peer claims to have the same id")
	}

	// Choose encryption params.
	var selectedCurve, selectedHash, selectedCipher string
	// We need everything to be perfomed the same way on both sides.
	order, err := utils.Compare(iden.Id, remoteId)
	if order > 0 {
		selectedCurve = selectParam(crypto.SupportedCurves, remoteInit.GetSupportedCurves())
		selectedHash = selectParam(crypto.SupportedHashes, remoteInit.GetSupportedHashes())
		selectedCipher = selectParam(crypto.SupportedCiphers, remoteInit.GetSupportedCiphers())
	} else {
		selectedCurve = selectParam(remoteInit.GetSupportedCurves(), crypto.SupportedCurves)
		selectedHash = selectParam(remoteInit.GetSupportedHashes(), crypto.SupportedHashes)
		selectedCipher = selectParam(remoteInit.GetSupportedCiphers(), crypto.SupportedCiphers)
	}

	if selectedCurve == "" || selectedHash == "" || selectedCipher == "" {
		return errors.New("Selection error")
	}

	//
	// === EXCHANGE HANDSHAKE MESSAGES ===
	//

	// Generate ephemeral key.
	curve, err := crypto.GetCurve(selectedCurve)
	if err != nil {
		return err
	}

	ephemeralKey, err := crypto.GenerateEphemeralKeypair(curve)
	if err != nil {
		return err
	}

	// Form Handshake message.
	localEphemeralKeyBytes, err := ephemeralKey.Bytes()
	if err != nil {
		return err
	}

	localHandshake := &message.Handshake{
		EphemeralPubKey: localEphemeralKeyBytes,
	}

	// Send Handshake message.
	err = p.SendWithContext(ctx, localHandshake)
	if err != nil {
		return err
	}

	// Receive Handshake message.
	msg, err = p.ReceiveWithContext(ctx)
	if err != nil {
		return err
	}
	remoteHandshake, ok := msg.(*message.Handshake)
	if !ok {
		return errors.New("Received message is not Handshake")
	}

	//
	// === PROCESS HANDSHAKE MESSAGES ===
	//

	// Generate shared secret.
	sharedSecret, err := ephemeralKey.GenerateSharedSecret(remoteHandshake.EphemeralPubKey)
	if err != nil {
		return err
	}

	// Generate two key pairs by stretching the secret.
	var salt []byte
	if order > 0 {
		salt = append(localNonce, remoteInit.GetNonce()...)
	} else {
		salt = append(remoteInit.GetNonce(), localNonce...)
	}
	k1, k2, err := crypto.StretchKey(sharedSecret, salt, selectedHash, selectedCipher)
	if order < 0 {
		k2, k1 = k1, k2
	}

	// Convert the byte nonces to initial integer nonces.
	var intLocalNonce, intRemoteNonce uint32
	bufRemoteNonce := bytes.NewBuffer(remoteInit.GetNonce())
	if err := binary.Read(bufRemoteNonce, binary.BigEndian, &intRemoteNonce); err != nil {
		return err
	}
	bufLocalNonce := bytes.NewBuffer(localNonce)
	if err := binary.Read(bufLocalNonce, binary.BigEndian, &intLocalNonce); err != nil {
		return err
	}

	// Initiate the secure encoder.
	layer, err := newSecure(k1, k2, intLocalNonce, intRemoteNonce, selectedHash, selectedCipher)
	if err != nil {
		return err
	}
	p.wrapper.AddLayer(layer)

	//
	// === EXCHANGE CONFIRMHANDSHAKE MESSAGES ===
	//

	hash, err := crypto.GetCryptoHash(selectedHash)
	if err != nil {
		return err
	}

	// Form ConfirmHandshake message.
	sig, err := iden.PrivKey.Sign(remoteInit.GetNonce(), hash) // TODO THIS IS A SERIOUS SECURITY BUG
	if err != nil {
		return err
	}

	localConfirm := &message.ConfirmHandshake{
		Nonce:     remoteInit.GetNonce(),
		Signature: sig,
	}

	// Send ConfirmHandshake message.
	err = p.SendWithContext(ctx, localConfirm)
	if err != nil {
		return err
	}

	// Receive ConfirmHandshake message.
	msg, err = p.ReceiveWithContext(ctx)
	if err != nil {
		return err
	}
	remoteConfirm, ok := msg.(*message.ConfirmHandshake)
	if !ok {
		return errors.New("Received message is not ConfirmHandshake")
	}

	//
	// === PROCESS CONFIRMHANDSHAKE MESSAGES ===
	//

	// Confirm identity.
	err = remotePub.Validate(localNonce, remoteConfirm.GetSignature(), hash)
	if err != nil {
		return err
	}

	confirm, err := utils.Compare(remoteConfirm.GetNonce(), localNonce)
	if err != nil || confirm != 0 {
		return errors.New("Received invalid nonce")
	}

	// Finally set up the peer.
	p.id = remoteId
	p.pubKey = remotePub

	return nil
}

func (p *peer) identify(ctx context.Context, listenAddr string) error {
	// Form Identity message.
	remoteAddr := p.conn.RemoteAddr().String()

	localIdentify := &message.Identity{
		ListenAddress:     &listenAddr,
		ConnectionAddress: &remoteAddr,
	}

	// Send Identity message.
	err := p.SendWithContext(ctx, localIdentify)
	if err != nil {
		return err
	}

	// Receive Identity message.
	msg, err := p.ReceiveWithContext(ctx)
	if err != nil {
		return err
	}
	remoteIdentify, ok := msg.(*message.Identity)
	if !ok {
		return errors.New("Received message is not Identity")
	}

	// Process Identity message.
	p.listenAddr = remoteIdentify.GetListenAddress()
	return nil
}
