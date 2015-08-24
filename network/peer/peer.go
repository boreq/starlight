package peer

import (
	"bytes"
	"errors"
	"github.com/boreq/netblog/crypto"
	"github.com/boreq/netblog/network/node"
	"github.com/boreq/netblog/protocol"
	"github.com/boreq/netblog/protocol/encoder"
	"github.com/boreq/netblog/protocol/message"
	"github.com/boreq/netblog/utils"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
	"io"
	"net"
	"reflect"
	"strings"
	"time"
)

type Peer interface {
	// Returns information about the node.
	Info() node.NodeInfo

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
		ctx:     ctx,
		cancel:  cancel,
		conn:    conn,
		encoder: encoder.NewBasic(),
	}

	p.in = make(chan []byte)
	go receiveFromPeer(p.ctx, p.in, p.conn, p.Close)
	p.out = make(chan []byte)
	go sendToPeer(p.ctx, p.out, p.conn)

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
	id         node.ID
	ctx        context.Context
	cancel     context.CancelFunc
	in         chan []byte
	out        chan []byte
	conn       net.Conn
	listenAddr string
	encoder    encoder.Encoder
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
	data, err := p.encoder.Encode(msg)
	if err != nil {
		return err
	}
	return p.send(data)
}

func (p *peer) SendWithContext(ctx context.Context, msg proto.Message) error {
	log.Debugf("%s sending %s: %s", p.id, reflect.TypeOf(msg), msg)
	data, err := p.encoder.Encode(msg)
	if err != nil {
		return err
	}
	return p.sendWithContext(ctx, data)
}

// send sends a raw message to the peer.
func (p *peer) send(data []byte) error {
	select {
	case p.out <- data:
		return nil
	case <-p.ctx.Done():
		return errors.New("Context closed, can not send")
	}
}

func (p *peer) sendWithContext(ctx context.Context, data []byte) error {
	select {
	case p.out <- data:
		return nil
	case <-ctx.Done():
		return errors.New("Passed context closed, can not send " + ctx.Err().Error())
	case <-p.ctx.Done():
		return errors.New("Context closed, can not send")
	}
}

func (p *peer) Receive() (proto.Message, error) {
	data, err := p.receive()
	if err != nil {
		return nil, err
	}
	return p.encoder.Decode(data)
}

func (p *peer) ReceiveWithContext(ctx context.Context) (proto.Message, error) {
	data, err := p.receiveWithContext(ctx)
	if err != nil {
		return nil, err
	}
	return p.encoder.Decode(data)
}

// receive receives a raw message from the peer.
func (p *peer) receive() ([]byte, error) {
	select {
	case data, ok := <-p.in:
		if !ok {
			return nil, errors.New("Channel closed, can't receive")
		}
		return data, nil
	case <-p.ctx.Done():
		return nil, errors.New("Context closed, can't receive")
	}
}

// receiveWithContext receives a raw message to the peer but returns with an
// error when ctx is closed.
func (p *peer) receiveWithContext(ctx context.Context) (data []byte, err error) {
	select {
	case data, ok := <-p.in:
		if !ok {
			return nil, errors.New("Channel closed, can't receive")
		}
		return data, nil
	case <-ctx.Done():
		return nil, errors.New("Passed context closed, can't receive" + ctx.Err().Error())
	case <-p.ctx.Done():
		return nil, errors.New("Context closed, can't receive")
	}
}

type cancelFunc func()

// Receives messages from a peer and sends them into the channel. In case of
// error the cancel func is called (which can for example close the underlying
// connnection or perform other cleanup).
func receiveFromPeer(ctx context.Context, in chan<- []byte, reader io.Reader, cancel cancelFunc) {
	unmarshaler := protocol.NewUnmarshaler(ctx, in)
	buf := make([]byte, 1024)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := reader.Read(buf)
			if err != nil {
				cancel()
				return
			}
			_, err = unmarshaler.Write(buf[:n])
			if err != nil {
				cancel()
				return
			}
		}
	}
}

// Reads messages from a channel and sends them to a peer.
func sendToPeer(ctx context.Context, out <-chan []byte, writer io.Writer) {
	buf := bytes.Buffer{}
	for {
		select {
		case <-ctx.Done():
			return
		case data := <-out:
			buf.Reset()
			data, err := protocol.Marshal(data)
			if err != nil {
				continue
			}
			buf.Write(data)
			buf.WriteTo(writer)
		}
	}
}

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
	data, err := p.encoder.Encode(localInit)
	if err != nil {
		return err
	}
	err = p.sendWithContext(ctx, data)
	if err != nil {
		return err
	}

	// Receive Init message.
	data, err = p.receiveWithContext(ctx)
	if err != nil {
		return err
	}
	msg, err := p.encoder.Decode(data)
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

	// Fail if the node id is the same.
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
	ephemeralKey, err := crypto.GenerateEphemeralKeypair(selectedCurve)
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
	data, err = p.encoder.Encode(localHandshake)
	if err != nil {
		return err
	}
	err = p.sendWithContext(ctx, data)
	if err != nil {
		return err
	}

	// Receive Handshake message.
	data, err = p.receiveWithContext(ctx)
	if err != nil {
		return err
	}
	msg, err = p.encoder.Decode(data)
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
	enc, err := encoder.NewSecure(k1, k2, selectedHash, selectedCipher)
	if err != nil {
		return err
	}

	//
	// === EXCHANGE CONFIRMHANDSHAKE MESSAGES ===
	//

	// Form ConfirmHandshake message.
	sig, err := iden.PrivKey.Sign(remoteInit.GetNonce(), selectedHash)
	if err != nil {
		return err
	}

	localConfirm := &message.ConfirmHandshake{
		Nonce:     remoteInit.GetNonce(),
		Signature: sig,
	}

	// Send ConfirmHandshake message.
	data, err = enc.Encode(localConfirm)
	if err != nil {
		return err
	}
	err = p.sendWithContext(ctx, data)
	if err != nil {
		return err
	}

	// Receive ConfirmHandshake message.
	data, err = p.receiveWithContext(ctx)
	if err != nil {
		return err
	}
	msg, err = enc.Decode(data)
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
	err = remotePub.Validate(localNonce, remoteConfirm.GetSignature(), selectedHash)
	if err != nil {
		return err
	}

	confirm, err := utils.Compare(remoteConfirm.GetNonce(), localNonce)
	if err != nil || confirm != 0 {
		return errors.New("Received invalid nonce")
	}

	// Finally set up the peer.
	p.id = remoteId
	p.encoder = enc

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
	data, err := p.encoder.Encode(localIdentify)
	if err != nil {
		return err
	}
	err = p.sendWithContext(ctx, data)
	if err != nil {
		return err
	}

	// Receive Identity message.
	data, err = p.receiveWithContext(ctx)
	if err != nil {
		return err
	}
	msg, err := p.encoder.Decode(data)
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
