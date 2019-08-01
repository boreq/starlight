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
	"github.com/boreq/starlight/network/stream"
	"github.com/boreq/starlight/utils"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

var log = utils.GetLogger("network")
var differentNodeIdError = errors.New("peer has a different id than requested")

const cleanupPeersEvery = 1 * time.Minute

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
	ctx        context.Context
	iden       node.Identity
	peers      []peer.Peer
	peersMutex sync.Mutex
	disp       dispatcher.Dispatcher
	nat        *natlib.NAT
	address    string
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

	// Periodically remove closed streams and peers that no longer have
	// open streams
	go func() {
		for {
			select {
			case <-time.After(cleanupPeersEvery):
				log.Debug("executing cleanupPeers")
				n.cleanupPeers()
			case <-n.ctx.Done():
				return
			}
		}
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
	s, err := stream.New(ctx, n.iden, n.getListeningAddresses(), conn)
	if err != nil {
		log.Debugf("newConnection: failed to init a stream: %s", err)
		return nil, errors.Wrap(err, "could not init a stream")
	}
	return n.newStream(s)
}

// Initiates a new peer.
func (n *network) newStream(s stream.Stream) (peer.Peer, error) {
	n.peersMutex.Lock()
	defer n.peersMutex.Unlock()

	// Add this stream to the appropriate peer (or create one)
	p, err := n.getPeerForStream(s)
	if err != nil {
		return nil, errors.Wrap(err, "could not get peer for stream")
	}

	// Receive and dispatch messages received via this stream
	go func() {
		for {
			msg, err := s.ReceiveWithContext(n.ctx)
			log.Debugf("%s received %T", s.Info().Id, msg)
			if err != nil && s.Closed() {
				log.Debugf("%s error %s, stopping the dispatcher loop", s.Info().Id, err)
				return
			}
			n.disp.Dispatch(s.Info(), msg)
		}
	}()

	log.Debugf("newStream: accepted %s reporting listener on %s ", s.Info().Id, s.Info().Address)

	return p, nil
}

const dialTimeout = 10 * time.Second

func (n *network) Dial(nd node.NodeInfo) (Peer, error) {
	log.Debugf("Dial: %s on %s", nd.Id, nd.Address)

	if node.CompareId(nd.Id, n.iden.Id) {
		return nil, errors.New("Tried calling a local id")
	}

	// Try to return an already existing peer
	n.peersMutex.Lock()
	p, err := n.findActive(nd.Id)
	n.peersMutex.Unlock()
	if err == nil {
		return p, nil
	}

	// Dial a peer if we are not already talking to it
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
	if !node.CompareId(nd.Id, p.Id()) {
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

	s, err := stream.New(ctx, n.iden, n.getListeningAddresses(), conn)
	if err != nil {
		return errors.Wrap(err, "could not create a peer")
	}

	if !node.CompareId(nd.Id, s.Info().Id) {
		return differentNodeIdError
	}

	go n.newStream(s)
	return nil
}

func (n *network) FindActive(id node.ID) (Peer, error) {
	n.peersMutex.Lock()
	defer n.peersMutex.Unlock()

	return n.findActive(id)
}

func (n *network) findActive(id node.ID) (peer.Peer, error) {
	for i := len(n.peers) - 1; i >= 0; i-- {
		if n.peers[i].Closed() {
			n.peers = append(n.peers[:i], n.peers[i+1:]...)
		} else {
			if node.CompareId(n.peers[i].Id(), id) {
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
	addresses := []string{n.address}
	if address, err := n.nat.GetAddress(); err != nil {
		log.Debugf("failed getting NAT address: %s", err)
	} else {
		log.Debugf("NAT address is: %s", address)
		addresses = append(addresses, address)
	}
	return addresses
}

func (n *network) getPeerForStream(s stream.Stream) (peer.Peer, error) {
	for _, p := range n.peers {
		if node.CompareId(s.Info().Id, p.Id()) {
			if err := p.AddStream(s); err != nil {
				return nil, errors.Wrap(err, "could not add the stream")
			}
			return p, nil
		}
	}
	p := peer.New(s)
	n.peers = append(n.peers, p)
	return p, nil
}

func (n *network) cleanupPeers() {
	n.peersMutex.Lock()
	defer n.peersMutex.Unlock()

	for i := len(n.peers) - 1; i >= 0; i-- {
		n.peers[i].Cleanup()
		if n.peers[i].Closed() {
			n.peers = append(n.peers[:i], n.peers[i+1:]...)
		}
	}
}
