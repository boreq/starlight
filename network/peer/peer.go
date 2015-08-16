package peer

import (
	"bytes"
	"errors"
	"github.com/boreq/netblog/crypto"
	"github.com/boreq/netblog/network/node"
	"github.com/boreq/netblog/protocol"
	"github.com/boreq/netblog/protocol/encoder"
	"github.com/boreq/netblog/protocol/message"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
	"io"
	"net"
	"time"
)

type Peer interface {
	// Returns information about the node.
	Info() node.NodeInfo

	// Sends a message to the node.
	Send(proto.Message) error

	// Receives a message from the node.
	Receive() (proto.Message, error)

	// Close ends communication with the node, closes the underlying
	// connection.
	Close()

	// Closed returns true if this peer has been closed.
	Closed() bool
}

// Use this instead of creating peer structs directly. Initiates communication
// channels and context.
func New(ctx context.Context, iden node.Identity, conn net.Conn) (Peer, error) {
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

	_, err := handshake(iden, p)
	if err != nil {
		p.Close()
		return nil, err
	}

	return p, nil
}

type peer struct {
	id      node.ID
	ctx     context.Context
	cancel  context.CancelFunc
	in      chan []byte
	out     chan []byte
	conn    net.Conn
	encoder encoder.Encoder
}

func (p *peer) Info() node.NodeInfo {
	return node.NodeInfo{
		Id:      p.id,
		Address: p.conn.RemoteAddr().String(),
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
	data, err := p.encoder.Encode(msg)
	if err != nil {
		return err
	}
	return p.send(data)
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

// sendWithContext sends a raw message to the peer but returns with an error
// when ctx is closed.
func (p *peer) sendWithContext(ctx context.Context, data []byte) error {
	var err error

	// This is basically a bug - but whatever, only used in handshake and
	// if this function times out, the handshake will fail, p.ctx will
	// be closed, p.send will fail and this function will end execution.
	c := make(chan error)
	go func() {
		err := p.send(data)
		select {
		case c <- err:
		case <-ctx.Done():
		}
	}()

	select {
	case err = <-c:
	case <-ctx.Done():
		return ctx.Err()
	}
	return err
}

func (p *peer) Receive() (proto.Message, error) {
	data, err := p.receive()
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
	// Again, this is a bug.
	c := make(chan error)
	go func() {
		data, err = p.receive()
		select {
		case c <- err:
		case <-ctx.Done():
		}
	}()

	select {
	case err = <-c:
	case <-ctx.Done():
		err = ctx.Err()
	}
	return
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
			unmarshaler.Write(buf[:n])
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

var handshakeErr = errors.New("Handshake error")
var handshakeTimeout = 5 * time.Second

// Performs a handshake and returns a secure encoder. Sets p.id.
func handshake(iden node.Identity, p *peer) (encoder.Encoder, error) {
	ctx, cancel := context.WithTimeout(p.ctx, handshakeTimeout)
	defer cancel()

	// Form Init message.
	ephemeralKey, err := crypto.GenerateEphemeralKeypair("P224")
	if err != nil {
		return nil, err
	}

	pubKeyBytes, err := iden.PubKey.Bytes()
	if err != nil {
		return nil, err
	}

	ephemeralKeyBytes, err := ephemeralKey.Bytes()
	if err != nil {
		return nil, err
	}

	localInit := &message.Init{
		PubKey:          pubKeyBytes,
		EphemeralPubKey: ephemeralKeyBytes,
	}

	// Send Init message.
	data, err := p.encoder.Encode(localInit)
	if err != nil {
		return nil, err
	}
	err = p.sendWithContext(ctx, data)
	if err != nil {
		return nil, err
	}

	// Receive Init message.
	data, err = p.receiveWithContext(ctx)
	if err != nil {
		return nil, err
	}
	msg, err := p.encoder.Decode(data)
	if err != nil {
		return nil, err
	}
	remoteInit, ok := msg.(*message.Init)
	if !ok {
		return nil, handshakeErr
	}

	// Process Init message.
	remotePub, err := crypto.NewPublicKey(remoteInit.PubKey)
	remoteId, err := remotePub.Hash()
	p.id = remoteId

	return encoder.NewBasic(), nil
}
