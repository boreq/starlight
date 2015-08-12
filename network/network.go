package network

import (
	"errors"
	"github.com/boreq/netblog/network/node"
	"github.com/boreq/netblog/utils"
	"golang.org/x/net/context"
	"net"
	"sync"
	"time"
)

var log = utils.Logger("network")

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
		log.Print("Context closed, closing listener")
		listener.Close()
	}()

	// Run the loop accepting connections.
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			log.Print("New incoming connection from %s", conn.RemoteAddr().String())
			go n.newConnection(n.ctx, conn)
		}
	}()

	return nil
}

// Initiates a new connection (incoming or outgoing).
func (n *network) newConnection(ctx context.Context, conn net.Conn) (*peer, error) {
	log.Print("newConnection")

	newP, err := newPeer(ctx, n.iden, conn)
	if err != nil {
		log.Print("newConnection: failed to init a peer", err)
		return nil, err
	}

	// If we are already communicating with this node, return the peer we
	// already have and terminate the new one.
	n.plock.Lock()
	defer n.plock.Unlock()
	p, err := n.findActive(newP.Id)
	if err == nil {
		log.Print("newConnection we already have this peer")
		newP.Close()
		return p, err
	}

	n.peers = append(n.peers, p)

	// Run dispatcher to be able to receive messages from all peers easily.
	go func() {
		for {
			msg, ok := <-newP.In()
			if !ok {
				log.Print("disp goroutine CHANNEL CLOSED")
				return
			}
			log.Print("disp goroutine dispatching")
			n.disp.Dispatch(newP, msg)
		}
	}()

	return newP, nil
}

func (n *network) Dial(nd node.NodeInfo) (Peer, error) {
	log.Printf("Dial: %s on %s", nd.Id, nd.Address)

	// Try to return an already existing peer.
	n.plock.Lock()
	p, err := n.findActive(nd.Id)
	n.plock.Unlock()
	if err == nil {
		log.Printf("Dial: already connected to %s", nd.Id)
		return p, nil
	}

	log.Printf("Dial: NOT already connected to %s", nd.Id)

	// Dial a peer if we are not already talking to it.
	conn, err := net.DialTimeout("tcp", nd.Address, 10*time.Second)
	if err != nil {
		log.Printf("Dial: failed to dial %s", nd.Id)
		return nil, err
	}
	p, err = n.newConnection(n.ctx, conn)
	if err != nil {
		log.Printf("Dial: failed to init connection %s", nd.Id)
		return nil, err
	}

	// Return an error if the id doesn't match but return the peer anyway.
	if !node.CompareId(nd.Id, p.Id) {
		log.Print("Dial: different node id, will warn")
		log.Printf("%x <-> %x", nd.Id, p.Id)
		return p, differentNodeIdError
	} else {
		log.Print("Dial: good node id")
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
