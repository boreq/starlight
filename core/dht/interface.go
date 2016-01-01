// Package dht implements a Kademlia based DHT. The DHT is used for storing
// public keys of the nodes participating in the network, channel memberships
// and routing.
package dht

import (
	"github.com/boreq/lainnet/crypto"
	"github.com/boreq/lainnet/network"
	"github.com/boreq/lainnet/network/dispatcher"
	"github.com/boreq/lainnet/network/node"
	"golang.org/x/net/context"
	"time"
)

type DHT interface {
	// Initializes the DHT using known bootstrap nodes.
	Init(nodes []node.NodeInfo) error

	// Ping sends a ping message to a node and returns the time which was
	// needed for the node to respond.
	Ping(ctx context.Context, id node.ID) (*time.Duration, error)

	// Dial attempts to return an already active Peer and if there is no
	// peer with the right id connected it attempts to locate and dial it.
	Dial(ctx context.Context, id node.ID) (network.Peer, error)

	// Subscribe returns a channel on which it is possible to receive
	// only validated incoming messages (signatures etc). CancelFunc must be
	// called afterwards.
	Subscribe() (chan dispatcher.IncomingMessage, dispatcher.CancelFunc)

	// FindNode attempts to locate a node and return its address.
	FindNode(ctx context.Context, id node.ID) (node.NodeInfo, error)

	// GetPubKey returns the public key of the specified node.
	GetPubKey(ctx context.Context, id node.ID) (crypto.PublicKey, error)

	// PutPubKey stores the public key of the specified node.
	PutPubKey(ctx context.Context, id node.ID, key crypto.PublicKey) error

	// GetChannel returns a list of nodes which have joined a channel.
	GetChannel(ctx context.Context, id []byte) ([]node.ID, error)

	// PutChannel stores the information about this node being in the
	// specifed channel. Other nodes can recover this information to know
	// which nodes should receive messages related to that channel.
	PutChannel(ctx context.Context, id []byte) error
}
