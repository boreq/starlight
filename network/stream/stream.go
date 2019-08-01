package stream

import (
	"net"
	"sync"
	"time"

	"github.com/boreq/starlight/crypto"
	"github.com/boreq/starlight/network/node"
	"github.com/boreq/starlight/protocol"
	"github.com/boreq/starlight/transport"
	"github.com/boreq/starlight/transport/basic"
	"github.com/boreq/starlight/utils"
	"github.com/boreq/starlight/utils/size"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

// handshakeTimeout specifies the time assigned for the handshake procedure. If
// the handshake takes longer the procedure will be aborted and the peer will
// be disconnected.
const handshakeTimeout = 5 * time.Second

// maxMessageSize specifies a max size of a message that is allowed to be sent
// between two peers. If a larger message is received the peer will be
// disconnected. If a larger message is attempted to be send the send action
// will fail.
const maxMessageSize = 100 * size.Kilobyte

var log = utils.GetLogger("stream")

// New attempts to create a new stream using the provided connection. During that
// process the handshake and other initialization will be performed. The
// function accepts the identity of the local node and the listen address of
// the local node (which is used to extract the port which the local node is
// listening on).
func New(ctx context.Context, iden node.Identity, listenAddresses []string, conn net.Conn) (Stream, error) {
	ctx, cancel := context.WithCancel(ctx)
	p := &stream{
		ctx:    ctx,
		cancel: cancel,
		conn:   conn,
	}
	p.wrapper = transport.NewWrapper(conn, conn)
	p.wrapper.AddLayer(basic.New(uint32(maxMessageSize)))

	hCtx, cancel := context.WithTimeout(p.ctx, handshakeTimeout)
	defer cancel()

	if err := p.handshake(hCtx, iden); err != nil {
		p.Close()
		log.Debugf("handshake error: %s", err)
		return nil, errors.Wrap(err, "handshake error")
	}

	if err := p.identify(hCtx, listenAddresses); err != nil {
		p.Close()
		log.Debugf("identify error: %s", err)
		return nil, errors.Wrap(err, "identify error")
	}

	return p, nil
}

type stream struct {
	id           node.ID
	pubKey       crypto.PublicKey
	ctx          context.Context
	cancel       context.CancelFunc
	conn         net.Conn
	listenAddr   []string
	wrapper      transport.Wrapper
	sendMutex    sync.Mutex
	receiveMutex sync.Mutex
}

func (p *stream) Info() node.NodeInfo {
	rHost, _, _ := net.SplitHostPort(p.conn.RemoteAddr().String())
	_, rPort, _ := net.SplitHostPort(p.getAppropriateListenAddr())
	address := net.JoinHostPort(rHost, rPort)

	return node.NodeInfo{
		Id:      p.id,
		Address: address,
	}
}

func (p *stream) getAppropriateListenAddr() string {
	remoteLocal, err := addrIsLocal(p.conn.RemoteAddr().String())
	if err == nil {
		for _, listenAddr := range p.listenAddr {
			reportedLocal, err := addrIsLocal(listenAddr)
			if err != nil {
				continue
			}

			if remoteLocal == reportedLocal {
				return listenAddr
			}
		}
	}

	if len(p.listenAddr) > 0 {
		return p.listenAddr[0]
	}
	return ""
}

func (p *stream) PubKey() crypto.PublicKey {
	return p.pubKey
}

func (p *stream) Closed() bool {
	select {
	case <-p.ctx.Done():
		return true
	default:
		return false
	}
}

func (p *stream) Close() {
	p.cancel()
	p.conn.Close()
}

func (p *stream) Send(msg proto.Message) error {
	data, err := protocol.Encode(msg)
	if err != nil {
		return errors.Wrap(err, "protocol encoding failed")
	}
	log.Debugf("%s sending %T (%d bytes)", p.id, msg, len(data))
	return p.send(data)
}

// send sends a raw message to the stream.
func (p *stream) send(data []byte) error {
	p.sendMutex.Lock()
	defer p.sendMutex.Unlock()
	err := p.wrapper.Send(data)
	if err != nil {
		log.Debugf("error on send %s, closing %s", err, p.id)
		p.Close()
	}
	return errors.Wrap(err, "wrapper send failed")
}

func (p *stream) SendWithContext(ctx context.Context, msg proto.Message) error {
	data, err := protocol.Encode(msg)
	if err != nil {
		return errors.Wrap(err, "protocol encoding failed")
	}
	log.Debugf("%s sending with context %T (%d bytes)", p.id, msg, len(data))
	return p.sendWithContext(ctx, data)
}

// sendWithContext attempts to send a raw message to the peer. Unfortunately
// this function is basically a bug and an ugly workaround around the way
// the encoder works - the goroutine that it launches will hang until
// the message is sent, so the function basically returns without confirming
// that the data has been sent instead of aborting completely.
func (p *stream) sendWithContext(ctx context.Context, data []byte) error {
	result := make(chan error)

	go func() {
		err := p.send(data)
		select {
		case result <- errors.Wrap(err, "send failed"):
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

func (p *stream) Receive() (proto.Message, error) {
	data, err := p.receive()
	if err != nil {
		return nil, errors.Wrap(err, "receive failed")
	}
	return protocol.Decode(data)
}

// receive receives a raw message from the peer.
func (p *stream) receive() ([]byte, error) {
	p.receiveMutex.Lock()
	defer p.receiveMutex.Unlock()
	data, err := p.wrapper.Receive()
	if err != nil {
		log.Debugf("error on receive %s, closing %s", err, p.id)
		p.Close()
	}
	return data, err
}

func (p *stream) ReceiveWithContext(ctx context.Context) (proto.Message, error) {
	data, err := p.receiveWithContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "receive with context failed")
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
func (p *stream) receiveWithContext(ctx context.Context) (data []byte, err error) {
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

var localHosts = []string{
	"",
	"127.0.0.1",
	"localhost",
}

func addrIsLocal(addr string) (bool, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false, err
	}

	for _, localHost := range localHosts {
		if host == localHost {
			return true, nil
		}
	}
	return false, nil
}
