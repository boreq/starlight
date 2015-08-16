package network

import (
	"errors"
	"github.com/boreq/netblog/network/node"
	"github.com/boreq/netblog/network/peer"
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
	peers []peer.Peer
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
func (n *network) newConnection(ctx context.Context, conn net.Conn) (peer.Peer, error) {
	p, err := peer.New(ctx, n.iden, conn)
	if err != nil {
		log.Print("newConnection: failed to init a peer", err)
		return nil, err
	}

	// If we are already communicating with this node, return the peer we
	// already have and terminate the new one.
	n.plock.Lock()
	defer n.plock.Unlock()
	existingPeer, err := n.findActive(p.Info().Id)
	if err == nil {
		log.Print("newConnection we already have this peer")
		p.Close()
		return existingPeer, err
	}

	n.peers = append(n.peers, p)

	// Run dispatcher to be able to receive messages from all peers easily.
	go func() {
		for {
			msg, err := p.Receive()
			if err != nil {
				log.Print("disp goroutine error", err)
				return
			}
			log.Print("disp goroutine dispatching")
			n.disp.Dispatch(p, msg)
		}
	}()

	log.Print("newConnection done with ", p.Info().Id)
	return p, nil
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
	if !node.CompareId(nd.Id, p.Info().Id) {
		log.Print("Dial: different node id, will warn")
		log.Printf("%s <-> %s", nd.Id, p.Info().Id)
		return p, differentNodeIdError
	} else {
		log.Print("Dial: good node id")
		return p, nil
	}
}

func (n *network) findActive(id node.ID) (peer.Peer, error) {
	for i := len(n.peers) - 1; i >= 0; i-- {
		if n.peers[i].Closed() {
			log.Print("dropped %s", n.peers[i].Info().Id)
			n.peers = append(n.peers[:i], n.peers[i+1:]...)
		} else {
			if node.CompareId(n.peers[i].Info().Id, id) {
				return n.peers[i], nil
			}
		}
	}

	return nil, errors.New("Peer not found")
}
