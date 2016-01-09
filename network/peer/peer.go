package peer

import (
	"crypto/cipher"
	"crypto/hmac"
	"errors"
	"github.com/boreq/lainnet/crypto"
	"github.com/boreq/lainnet/network/node"
	"github.com/boreq/lainnet/protocol"
	"github.com/boreq/lainnet/protocol/message"
	"github.com/boreq/lainnet/transport"
	"github.com/boreq/lainnet/transport/basic"
	"github.com/boreq/lainnet/transport/secure"
	"github.com/boreq/lainnet/utils"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
	"io"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"
)

type Peer interface {
	// Returns information about the node.
	Info() node.NodeInfo

	// Returns the node's public key.
	PubKey() crypto.PublicKey

	// Sends a message to the node.
	Send(proto.Message) error

	// Sends a message to the node, returns an error if context is closed.
	SendWithContext(context.Context, proto.Message) error

	// Receives a message from the node.
	Receive() (proto.Message, error)

	// Receives a message from the node, returns an error if context is
	// closed.
	ReceiveWithContext(context.Context) (proto.Message, error)

	// Close ends communication with the node, closes the underlying
	// connection.
	Close()

	// Closed returns true if this peer has been closed.
	Closed() bool
}

var handshakeTimeout = 5 * time.Second
var log = utils.GetLogger("peer")

// Use this instead of creating peer structs directly. Initiates communication
// channels and context.
func New(ctx context.Context, iden node.Identity, listenAddress string, conn net.Conn) (Peer, error) {
	ctx, cancel := context.WithCancel(ctx)
	p := &peer{
		ctx:    ctx,
		cancel: cancel,
		conn:   conn,
	}
	p.encoder, p.decoder = basic.New(conn)

	hCtx, cancel := context.WithTimeout(p.ctx, handshakeTimeout)
	defer cancel()

	err := p.handshake(hCtx, iden)
	if err != nil {
		log.Debug("HANDSHAKE ERROR")
		p.Close()
		return nil, err
	}
	err = p.identify(hCtx, listenAddress)
	if err != nil {
		log.Debug("IDENTIFY ERROR")
		p.Close()
		return nil, err
	}

	return p, nil
}

type peer struct {
	id           node.ID
	pubKey       crypto.PublicKey
	ctx          context.Context
	cancel       context.CancelFunc
	conn         net.Conn
	listenAddr   string
	encoder      transport.Encoder
	encoderMutex sync.Mutex
	decoder      transport.Decoder
	decoderMutex sync.Mutex
}

func (p *peer) Info() node.NodeInfo {
	rHost, _, _ := net.SplitHostPort(p.conn.RemoteAddr().String())
	_, rPort, _ := net.SplitHostPort(p.listenAddr)
	address := net.JoinHostPort(rHost, rPort)

	return node.NodeInfo{
		Id:      p.id,
		Address: address,
	}
}

func (p *peer) PubKey() crypto.PublicKey {
	return p.pubKey
}

func (p *peer) Closed() bool {
	select {
	case <-p.ctx.Done():
		return true
	default:
		return false
	}
}

func (p *peer) Close() {
	p.cancel()
	p.conn.Close()
}

func (p *peer) Send(msg proto.Message) error {
	log.Debugf("%s sending %s: %s", p.id, reflect.TypeOf(msg), msg)
	data, err := protocol.Encode(msg)
	if err != nil {
		return err
	}
	return p.send(data)
}

func (p *peer) SendWithContext(ctx context.Context, msg proto.Message) error {
	log.Debugf("%s sending %s: %s", p.id, reflect.TypeOf(msg), msg)
	data, err := protocol.Encode(msg)
	if err != nil {
		return err
	}
	return p.sendWithContext(ctx, data)
}

// send sends a raw message to the peer.
func (p *peer) send(data []byte) error {
	p.encoderMutex.Lock()
	defer p.encoderMutex.Unlock()
	return p.encoder.Encode(data)
}

func (p *peer) sendWithContext(ctx context.Context, data []byte) error {
	// TODO
	return p.send(data)
}

func (p *peer) Receive() (proto.Message, error) {
	data, err := p.receive()
	if err != nil {
		return nil, err
	}
	return protocol.Decode(data)
}

func (p *peer) ReceiveWithContext(ctx context.Context) (proto.Message, error) {
	data, err := p.receiveWithContext(ctx)
	if err != nil {
		return nil, err
	}
	return protocol.Decode(data)
}

// receive receives a raw message from the peer.
func (p *peer) receive() ([]byte, error) {
	p.decoderMutex.Lock()
	defer p.decoderMutex.Unlock()
	return p.decoder.Decode()
}

// receiveWithContext receives a raw message to the peer but returns with an
// error when ctx is closed.
func (p *peer) receiveWithContext(ctx context.Context) (data []byte, err error) {
	// TODO
	return p.receive()
}

type cancelFunc func()

// selectParam
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

func newSecure(rw io.ReadWriter, localKeys, remoteKeys crypto.StretchedKeys, hashName string, cipherName string) (transport.Encoder, transport.Decoder, error) {
	hash, err := crypto.GetCryptoHash(hashName)
	if err != nil {
		return nil, nil, err
	}

	localCipher, err := crypto.GetCipher(cipherName, localKeys.CipherKey)
	if err != nil {
		return nil, nil, err
	}

	remoteCipher, err := crypto.GetCipher(cipherName, remoteKeys.CipherKey)
	if err != nil {
		return nil, nil, err
	}

	encHmac := hmac.New(hash.New, localKeys.MacKey)
	decHmac := hmac.New(hash.New, remoteKeys.MacKey)
	encCipher := cipher.NewCBCEncrypter(localCipher, localKeys.IV)
	decCipher := cipher.NewCBCDecrypter(remoteCipher, remoteKeys.IV)
	enc, dec := secure.New(rw, decHmac, encHmac, decCipher, encCipher)
	return enc, dec, nil
}

var nonceSize = 20

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

	// Initiate the secure encoder.
	p.encoder, p.decoder, err = newSecure(p.conn, k1, k2, selectedHash, selectedCipher)
	if err != nil {
		return err
	}

	//
	// === EXCHANGE CONFIRMHANDSHAKE MESSAGES ===
	//

	hash, err := crypto.GetCryptoHash(selectedHash)
	if err != nil {
		return err
	}

	// Form ConfirmHandshake message.
	sig, err := iden.PrivKey.Sign(remoteInit.GetNonce(), hash)
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
