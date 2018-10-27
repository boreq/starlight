package peer

import (
	"github.com/boreq/starlight/crypto"
	"github.com/boreq/starlight/network/node"
	"github.com/boreq/starlight/protocol"
	"github.com/boreq/starlight/transport"
	"github.com/boreq/starlight/transport/basic"
	"github.com/boreq/starlight/utils"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
	"net"
	"sync"
	"time"
)

// handshakeTimeout specifies the time assigned for the handshake procedure. If
// the handshake takes longer the procedure will be aborted and the peer will
// be disconnected.
const handshakeTimeout = 5 * time.Second

var log = utils.GetLogger("peer")

// New attempts to create a new peer using the provided connection. During that
// process the handshake and other initialization will be performed. The
// function accepts the identity of the local node and the listen address of
// the local node (which is used to extract the port which the local node is
// listening on).
func New(ctx context.Context, iden node.Identity, listenAddress string, conn net.Conn) (Peer, error) {
	ctx, cancel := context.WithCancel(ctx)
	p := &peer{
		ctx:    ctx,
		cancel: cancel,
		conn:   conn,
	}
	p.wrapper = transport.NewWrapper(conn, conn)
	p.wrapper.AddLayer(basic.New())

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
	wrapper      transport.Wrapper
	sendMutex    sync.Mutex
	receiveMutex sync.Mutex
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
	log.Debugf("%s sending %T", p.id, msg)
	data, err := protocol.Encode(msg)
	if err != nil {
		return err
	}
	return p.send(data)
}

// send sends a raw message to the peer.
func (p *peer) send(data []byte) error {
	p.sendMutex.Lock()
	defer p.sendMutex.Unlock()
	err := p.wrapper.Send(data)
	if err != nil {
		log.Debugf("error on send %s, closing %s", err, p.id)
		p.Close()
	}
	return err
}

func (p *peer) SendWithContext(ctx context.Context, msg proto.Message) error {
	log.Debugf("%s sending with context %T", p.id, msg)
	data, err := protocol.Encode(msg)
	if err != nil {
		return err
	}
	return p.sendWithContext(ctx, data)
}

// sendWithContext attempts to send a raw message to the peer. Unfortunately
// this function is basically a bug and an ugly workaround around the way
// the encoder works - the goroutine that it launches will hang until
// the message is sent, so the function basically returns without confirming
// that the data has been sent instead of aborting completely.
func (p *peer) sendWithContext(ctx context.Context, data []byte) error {
	result := make(chan error)

	go func() {
		err := p.send(data)
		select {
		case result <- err:
		case <-ctx.Done():
		}
	}()

	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *peer) Receive() (proto.Message, error) {
	data, err := p.receive()
	if err != nil {
		return nil, err
	}
	return protocol.Decode(data)
}

// receive receives a raw message from the peer.
func (p *peer) receive() ([]byte, error) {
	p.receiveMutex.Lock()
	defer p.receiveMutex.Unlock()
	data, err := p.wrapper.Receive()
	if err != nil {
		log.Debugf("error on receive %s, closing %s", err, p.id)
		p.Close()
	}
	return data, err
}

func (p *peer) ReceiveWithContext(ctx context.Context) (proto.Message, error) {
	data, err := p.receiveWithContext(ctx)
	if err != nil {
		return nil, err
	}
	return protocol.Decode(data)
}

// receiveWithContext attempts to receive a raw message from the peer.
// This method is an ugly workaround as well - the goroutine that it launches
// will not terminate until a message is received. That message will be lost if
// the method returns earlier. Fortunately this is not an issue because of the
// way the rest of the program is structured - everything is received in the
// same loop and if the context closes then the peer will be disconnected anyway
// because it will either be during the handshake or because of the serious
// protocol error which disconnects the the peer.
func (p *peer) receiveWithContext(ctx context.Context) (data []byte, err error) {
	result := make(chan []byte)
	resultErr := make(chan error)

	go func() {
		data, err := p.receive()
		if err != nil {
			select {
			case resultErr <- err:
			case <-ctx.Done():
			}
		} else {
			select {
			case result <- data:
			case <-ctx.Done():
			}
		}
	}()

	select {
	case data := <-result:
		return data, nil
	case err := <-resultErr:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
