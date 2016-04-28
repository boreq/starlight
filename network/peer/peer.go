package peer

import (
	"github.com/boreq/lainnet/crypto"
	"github.com/boreq/lainnet/network/node"
	"github.com/boreq/lainnet/protocol"
	"github.com/boreq/lainnet/transport"
	"github.com/boreq/lainnet/transport/basic"
	"github.com/boreq/lainnet/utils"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
	"net"
	"reflect"
	"sync"
	"time"
)

const handshakeTimeout = 5 * time.Second

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
