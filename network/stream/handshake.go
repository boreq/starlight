package stream

import (
	"bytes"
	"crypto/cipher"
	"crypto/hmac"
	"encoding/binary"
	"strings"

	"github.com/boreq/starlight/crypto"
	"github.com/boreq/starlight/network/node"
	"github.com/boreq/starlight/protocol/message"
	"github.com/boreq/starlight/transport"
	"github.com/boreq/starlight/transport/secure"
	"github.com/boreq/starlight/utils"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
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

// sendAsyncWithContext prevents a deadlock when mutually exchanging
// synchronous messages with another node. For example during a handshake we
// want to send a message but at the same time receive an incoming message so
// the message must be send at the same time as we are waiting to receive the
// incomming message. As the messages are small this could work due to OS
// buffering and the way the network stack works but it is a dangerous gamble.
func (p *stream) sendAsyncWithContext(ctx context.Context, msg proto.Message) <-chan error {
	errC := make(chan error)
	go func() {
		err := p.SendWithContext(ctx, msg)
		if err != nil {
			errC <- errors.Wrap(err, "could not send with context")
		}
	}()
	return errC
}

// receiveAsyncWithContext is a conterpart of sendAsyncWithContext which makes
// it easy to perform receive and send operations at the same time by using a
// select on their results.
func (p *stream) receiveAsyncWithContext(ctx context.Context) (<-chan proto.Message, <-chan error) {
	errC := make(chan error)
	msgC := make(chan proto.Message)
	go func() {
		msg, err := p.ReceiveWithContext(ctx)
		if err != nil {
			errC <- errors.Wrap(err, "could not receive with context")
			return
		}
		msgC <- msg
	}()
	return msgC, errC
}

func (p *stream) exchangeMessages(ctx context.Context, msg proto.Message) (proto.Message, error) {
	sendErrC := p.sendAsyncWithContext(ctx, msg)
	msgC, recErrC := p.receiveAsyncWithContext(ctx)
	select {
	case err := <-sendErrC:
		return nil, errors.Wrap(err, "async send failed")
	case err := <-recErrC:
		return nil, errors.Wrap(err, "async receive failed")
	case msg := <-msgC:
		return msg, nil
	}
}

// The ride never ends. Performs a handshake, sets up a secure encoder and peer
// id.
func (p *stream) handshake(ctx context.Context, iden node.Identity) error {

	//
	// === EXCHANGE INIT MESSAGES ===
	//

	// Form an Init message
	pubKeyBytes, err := iden.PubKey.Bytes()
	if err != nil {
		return errors.Wrap(err, "could not get the pub key bytes")
	}

	localNonce := make([]byte, nonceSize)
	err = crypto.GenerateNonce(localNonce)
	if err != nil {
		return errors.Wrap(err, "could not generate a nonce")
	}

	localInit := &message.Init{
		PubKey:           pubKeyBytes,
		Nonce:            localNonce,
		SupportedCurves:  &crypto.SupportedCurves,
		SupportedHashes:  &crypto.SupportedHashes,
		SupportedCiphers: &crypto.SupportedCiphers,
	}

	// Exchange Init messages
	msg, err := p.exchangeMessages(ctx, localInit)
	if err != nil {
		return errors.Wrap(err, "could not exchange init messages")
	}
	remoteInit, ok := msg.(*message.Init)
	if !ok {
		return errors.New("the received message is not Init")
	}

	//
	// === PROCESS INIT MESSAGES ===
	//

	// Establish identity
	remotePub, err := crypto.NewPublicKey(remoteInit.GetPubKey())
	if err != nil {
		return errors.Wrap(err, "could not create the remote public key")
	}
	remoteId, err := remotePub.Hash()
	if err != nil {
		return errors.Wrap(err, "could not hash the remote public key")
	}

	// Fail if the id is invalid
	if !node.ValidateId(remoteId) {
		return errors.New("invalid remote id")
	}

	// Fail if the id is the same as the id of the local node
	if node.CompareId(remoteId, iden.Id) {
		return errors.New("peer claims to have the same id as the local node")
	}

	// Choose encryption params
	var selectedCurve, selectedHash, selectedCipher string
	// We need everything to be perfomed the same way on both sides.
	order, err := utils.Compare(iden.Id, remoteId)
	if err != nil {
		return err
	}
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
		return errors.New("selection error")
	}

	//
	// === EXCHANGE HANDSHAKE MESSAGES ===
	//

	// Generate ephemeral key
	curve, err := crypto.GetCurve(selectedCurve)
	if err != nil {
		return errors.Wrap(err, "could not get a selected curve")
	}

	ephemeralKey, err := crypto.GenerateEphemeralKeypair(curve)
	if err != nil {
		return errors.Wrap(err, "could not generate an ephemeral keypair")
	}

	// Form Handshake message
	localEphemeralKeyBytes, err := ephemeralKey.Bytes()
	if err != nil {
		return errors.Wrap(err, "could not get local ephemeral key bytes")
	}

	localHandshake := &message.Handshake{
		EphemeralPubKey: localEphemeralKeyBytes,
	}

	// Exchange Handshake messages
	msg, err = p.exchangeMessages(ctx, localHandshake)
	if err != nil {
		return errors.Wrap(err, "could not exchange handshake messages")
	}
	remoteHandshake, ok := msg.(*message.Handshake)
	if !ok {
		return errors.New("received message is not Handshake")
	}

	//
	// === PROCESS HANDSHAKE MESSAGES ===
	//

	// Generate shared secret.
	sharedSecret, err := ephemeralKey.GenerateSharedSecret(remoteHandshake.EphemeralPubKey)
	if err != nil {
		return err
	}

	// Generate two key pairs by stretching the secret
	var salt []byte
	if order > 0 {
		salt = append(localNonce, remoteInit.GetNonce()...)
	} else {
		salt = append(remoteInit.GetNonce(), localNonce...)
	}
	k1, k2, err := crypto.StretchKey(sharedSecret, salt, selectedHash, selectedCipher)
	if err != nil {
		return errors.Wrap(err, "key stretching failed")
	}
	if order < 0 {
		k2, k1 = k1, k2
	}

	// Convert the byte nonces to initial integer nonces
	var intLocalNonce, intRemoteNonce uint32
	bufRemoteNonce := bytes.NewBuffer(remoteInit.GetNonce())
	if err := binary.Read(bufRemoteNonce, binary.BigEndian, &intRemoteNonce); err != nil {
		return errors.Wrap(err, "could not read the remote nonce")
	}
	bufLocalNonce := bytes.NewBuffer(localNonce)
	if err := binary.Read(bufLocalNonce, binary.BigEndian, &intLocalNonce); err != nil {
		return errors.Wrap(err, "could not read the local nonce")
	}

	// Initiate the secure encoder
	layer, err := newSecure(k1, k2, intLocalNonce, intRemoteNonce, selectedHash, selectedCipher)
	if err != nil {
		return errors.Wrap(err, "could not create a secure encoder")
	}
	p.wrapper.AddLayer(layer)

	//
	// === EXCHANGE CONFIRMHANDSHAKE MESSAGES ===
	//

	hash, err := crypto.GetCryptoHash(selectedHash)
	if err != nil {
		return err
	}

	// Create values to be signed.
	valueToSign := bytes.Buffer{}
	if order > 0 {
		valueToSign.Write(localInit.GetNonce())
		valueToSign.Write(localInit.GetPubKey())
		valueToSign.Write(remoteInit.GetNonce())
		valueToSign.Write(remoteInit.GetPubKey())
	} else {
		valueToSign.Write(remoteInit.GetNonce())
		valueToSign.Write(remoteInit.GetPubKey())
		valueToSign.Write(localInit.GetNonce())
		valueToSign.Write(localInit.GetPubKey())
	}
	valueToSign.WriteString(selectedCurve)
	valueToSign.WriteString(selectedHash)
	valueToSign.WriteString(selectedCipher)

	// Form ConfirmHandshake message.
	sig, err := iden.PrivKey.Sign(valueToSign.Bytes(), hash)
	if err != nil {
		return errors.Wrap(err, "signing with the private key failed")
	}

	localConfirm := &message.ConfirmHandshake{
		Nonce:     remoteInit.GetNonce(),
		Signature: sig,
	}

	// Exchange ConfirmHandshake messages.
	msg, err = p.exchangeMessages(ctx, localConfirm)
	if err != nil {
		return errors.Wrap(err, "could not exchange confirm hanshake messages")
	}
	remoteConfirm, ok := msg.(*message.ConfirmHandshake)
	if !ok {
		return errors.New("received message is not ConfirmHandshake")
	}

	//
	// === PROCESS CONFIRMHANDSHAKE MESSAGES ===
	//

	// Confirm identity
	err = remotePub.Validate(valueToSign.Bytes(), remoteConfirm.GetSignature(), hash)
	if err != nil {
		return errors.Wrap(err, "remote identity confirmation failed")
	}

	confirm, err := utils.Compare(remoteConfirm.GetNonce(), localNonce)
	if err != nil || confirm != 0 {
		return errors.New("received an invalid nonce")
	}

	// Finally set up the peer.
	p.id = remoteId
	p.pubKey = remotePub

	return nil
}

func (p *stream) identify(ctx context.Context, listenAddresses []string) error {
	// Form Identity message.
	remoteAddr := p.conn.RemoteAddr().String()

	localIdentify := &message.Identity{
		ListenAddresses:   listenAddresses,
		ConnectionAddress: &remoteAddr,
	}

	// Exchange Identity messages.
	msg, err := p.exchangeMessages(ctx, localIdentify)
	if err != nil {
		return errors.Wrap(err, "could not exchange identify messages")
	}
	remoteIdentify, ok := msg.(*message.Identity)
	if !ok {
		return errors.New("Received message is not Identity")
	}

	// Process Identity message.
	p.listenAddr = remoteIdentify.GetListenAddresses()
	return nil
}
