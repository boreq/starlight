package network

import (
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/boreq/starlight/network/dispatcher"
	natlib "github.com/boreq/starlight/network/nat"
	"github.com/boreq/starlight/network/node"
	"github.com/boreq/starlight/network/peer"
	"github.com/boreq/starlight/utils"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

var log = utils.GetLogger("network")
var differentNodeIdError = errors.New("peer has a different id than requested")

func New(ctx context.Context, ident node.Identity, address string) Network {
	net := &network{
		ctx:     ctx,
		iden:    ident,
		disp:    dispatcher.New(ctx),
		address: address,
	}
	return net
}

type network struct {
	ctx     context.Context
	iden    node.Identity
	peers   []peer.Peer
	plock   sync.Mutex
	disp    dispatcher.Dispatcher
	nat     *natlib.NAT
	address string
}

func (n *network) Subscribe() (chan dispatcher.IncomingMessage, dispatcher.CancelFunc) {
	return n.disp.Subscribe()
}

func (n *network) Listen() error {
	log.Printf("Listening on %s", n.address)
	log.Printf("Local id %s", n.iden.Id)

	// Start listening
	listener, err := net.Listen("tcp", n.address)
	if err != nil {
		return errors.Wrap(err, "could not listen")
	}

	// Initialize the NAT traversal
	if err := n.initNatTraversal(); err != nil {
		return errors.Wrap(err, "could not init the NAT traversal")
	}

	// Make sure to close the listener after the context is closed
	go func() {
		<-n.ctx.Done()
		listener.Close()
	}()

	// Run the loop accepting connections
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			log.Debugf("New incoming connection from %s", conn.RemoteAddr().String())
			go n.newConnection(n.ctx, conn)
		}
	}()

	return nil
}

// Initiates a new connection (incoming or outgoing).
func (n *network) newConnection(ctx context.Context, conn net.Conn) (peer.Peer, error) {
	p, err := peer.New(ctx, n.iden, n.getListeningAddresses(), conn)
	if err != nil {
		log.Debugf("newConnection: failed to init a peer: %s", err)
		return nil, errors.Wrap(err, "could not init a peer")
	}
	return n.newPeer(p)
}

// Initiates a new peer.
func (n *network) newPeer(p peer.Peer) (peer.Peer, error) {
	// If we are already communicating with this node, return the peer we
	// already have and terminate the new one.
	n.plock.Lock()
	defer n.plock.Unlock()
	existingPeer, err := n.findActive(p.Info().Id)
	if err == nil {
		log.Debugf("newConnection: already communicating with %x", p.Info().Id)
		p.Close()
		return existingPeer, nil
	}

	n.peers = append(n.peers, p)

	// Run a dispatcher to be able to receive messages from all peers easily.
	go func() {
		for {
			msg, err := p.ReceiveWithContext(n.ctx)
			log.Debugf("%s received %T", p.Info().Id, msg)
			if err != nil && p.Closed() {
				log.Debugf("%s error %s, stopping the dispatcher loop", p.Info().Id, err)
				return
			}
			n.disp.Dispatch(p.Info(), msg)
		}
	}()

	log.Debugf("newConnection: accepted %s reporting listener on %s ", p.Info().Id, p.Info().Address)

	return p, nil
}

const dialTimeout = 10 * time.Second

func (n *network) Dial(nd node.NodeInfo) (Peer, error) {
	log.Debugf("Dial: %s on %s", nd.Id, nd.Address)

	if node.CompareId(nd.Id, n.iden.Id) {
		return nil, errors.New("Tried calling a local id")
	}

	// Try to return an already existing peer.
	n.plock.Lock()
	p, err := n.findActive(nd.Id)
	n.plock.Unlock()
	if err == nil {
		return p, nil
	}

	// Dial a peer if we are not already talking to it.
	conn, err := net.DialTimeout("tcp", nd.Address, dialTimeout)
	if err != nil {
		log.Debug("Dial: not responding", err)
		return nil, err
	}

	p, err = n.newConnection(n.ctx, conn)
	if err != nil {
		log.Debug("Dial: failed to init connection", err)
		return nil, err
	}

	// Return an error if the id doesn't match but return the peer anyway.
	if !node.CompareId(nd.Id, p.Info().Id) {
		log.Debug("Dial: different node id, will warn")
		return p, differentNodeIdError
	} else {
		return p, nil
	}
}

func (n *network) CheckOnline(ctx context.Context, nd node.NodeInfo) error {
	log.Debugf("CheckOnline: %s on %s", nd.Id, nd.Address)

	if node.CompareId(nd.Id, n.iden.Id) {
		return errors.New("tried checking a local id")
	}

	conn, err := net.DialTimeout("tcp", nd.Address, dialTimeout)
	if err != nil {
		return errors.Wrap(err, "could not dial")
	}

	p, err := peer.New(ctx, n.iden, n.getListeningAddresses(), conn)
	if err != nil {
		return errors.Wrap(err, "could not create a peer")
	}

	if !node.CompareId(nd.Id, p.Info().Id) {
		return differentNodeIdError
	}

	go n.newPeer(p)
	return nil
}

func (n *network) FindActive(id node.ID) (Peer, error) {
	return n.findActive(id)
}

func (n *network) findActive(id node.ID) (peer.Peer, error) {
	for i := len(n.peers) - 1; i >= 0; i-- {
		if n.peers[i].Closed() {
			n.peers = append(n.peers[:i], n.peers[i+1:]...)
		} else {
			if node.CompareId(n.peers[i].Info().Id, id) {
				return n.peers[i], nil
			}
		}
	}
	return nil, errors.New("Peer not found")
}

func (n *network) initNatTraversal() error {
	internalListeningPort, err := n.getInternalListeningPort()
	if err != nil {
		return errors.Wrap(err, "could not get the listening port")
	}

	nat, err := natlib.New(n.ctx, internalListeningPort)
	if err != nil {
		return errors.Wrap(err, "could not establish NAT piercing")
	}
	n.nat = nat
	return nil
}

func (n *network) getInternalListeningPort() (int, error) {
	_, port, err := net.SplitHostPort(n.address)
	if err != nil {
		return 0, errors.Wrap(err, "split host port failed")
	}
	return strconv.Atoi(port)
}

func (n *network) getListeningAddresses() []string {
	var addresses []string
	addresses = append(addresses, n.address)
	if address, err := n.nat.GetAddress(); err != nil {
		log.Debugf("get listening addresses error: %s", err)
	} else {
		log.Debugf("external listening addresses is: %s", address)
		addresses = append(addresses, address)
	}
	return addresses
}
