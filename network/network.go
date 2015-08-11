package network

import (
	"errors"
	"github.com/boreq/netblog/network/node"
	"github.com/boreq/netblog/protocol"
	"github.com/boreq/netblog/protocol/message"
	"golang.org/x/net/context"
	"net"
	"sync"
	"time"
)

var differentNodeIdError = errors.New("Peer under this address has a different id than requested")

func New(ctx context.Context, ident node.Identity) Network {
	net := &network{
		ctx:  ctx,
		iden: ident,
		disp: NewDispatcher(ctx),
	}
	return net
}

type network struct {
	ctx   context.Context
	iden  node.Identity
	peers []*peer
	plock sync.Mutex
	disp  Dispatcher
}

func (n *network) Subscribe() (chan IncomingMessage, CancelFunc) {
	return n.disp.Subscribe()
}

func (n *network) Listen(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	// Make sure to close the listener after the context is closed.
	go func() {
		<-n.ctx.Done()
		listener.Close()
	}()

	// Run the loop accepting connections.
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go n.newConnection(n.ctx, conn)
		}
	}()

	return nil
}

// Initiates a new connection (incoming or outgoing).
func (n *network) newConnection(ctx context.Context, conn net.Conn) (*peer, error) {
	p := newPeer(ctx, conn)

	// Perform hanshake.
	err := handshake(n.iden, p)
	if err != nil {
		p.Close()
		return nil, err
	}

	// Add peer to the list, if we are already talking terminate the
	// connection.
	n.plock.Lock()
	defer n.plock.Unlock()
	_, err = n.findActive(p.Id)
	if err == nil {
		p.Close()
		return nil, err
	}
	n.peers = append(n.peers, p)

	// Run dispatcher to be able to receive messages from all peers easily.
	go func() {
		for {
			select {
			case <-p.ctx.Done():
				return
			case msg := <-p.In:
				n.disp.Dispatch(p, msg)
			}
		}
	}()

	return p, nil
}

func (n *network) Dial(nd node.NodeInfo) (Peer, error) {
	// Try to return an already existing peer.
	n.plock.Lock()
	p, err := n.findActive(nd.Id)
	n.plock.Unlock()
	if err == nil {
		return p, nil
	}

	// Dial a peer if we are not already talking to it.
	conn, err := net.DialTimeout("tcp", nd.Address, 10*time.Second)
	if err != nil {
		return nil, err
	}
	p, err = n.newConnection(n.ctx, conn)
	if err != nil {
		return nil, err
	}

	// Return an error if the id doesn't match but return the peer anyway.
	if !node.CompareId(nd.Id, p.Id) {
		return p, differentNodeIdError
	} else {
		return p, nil
	}
}

func (n *network) findActive(id node.ID) (*peer, error) {
	for _, p := range n.peers {
		if node.CompareId(p.Id, id) {
			return p, nil
		}
	}
	return nil, errors.New("Peer not found")
}

func handshake(iden node.Identity, p *peer) error {
	pubKeyBytes, err := iden.PubKey.Bytes()
	if err != nil {
		return err
	}

	init := &message.Init{
		PubKey: pubKeyBytes,
	}
	msg, err := protocol.Marshal(protocol.Init, init)
	if err != nil {
		return err
	}

	p.Out <- *msg
	return nil
}
