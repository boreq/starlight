package network

import (
	"github.com/boreq/starlight/crypto"
	"github.com/boreq/starlight/network/dispatcher"
	"github.com/boreq/starlight/network/node"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
)

// Network is used to exchange messages with other nodes participating in the
// network.
type Network interface {
	// Listen starts listening on the given address, does not block.
	Listen() error

	// Dial returns an already connected Peer or if the connection does not
	// exist attempts to establish it.
	Dial(node node.NodeInfo) (Peer, error)

	// CheckOnline checks if the node is available under the specified
	// address.
	CheckOnline(ctx context.Context, node node.NodeInfo) error

	// FindActive returns an already connected Peer.
	FindActive(id node.ID) (Peer, error)

	// Subscribe returns a channel on which it is possible to receive all
	// incoming messages. CancelFunc must be called afterwards.
	Subscribe() (chan dispatcher.IncomingMessage, dispatcher.CancelFunc)
}

// Peer represents an external node.
type Peer interface {
	// Id returns the id of this peer.
	Id() node.ID

	// Returns the node's public key.
	PubKey() crypto.PublicKey

	// Sends a message to the node.
	Send(proto.Message) error

	// Sends a message to the node, returns an error if context is closed.
	SendWithContext(context.Context, proto.Message) error
}
