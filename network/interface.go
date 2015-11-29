package network

import (
	"github.com/boreq/lainnet/crypto"
	"github.com/boreq/lainnet/network/node"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
)

// Since all incoming messages are passed on the same channel they must be
// bundled with a node.ID.
type IncomingMessage struct {
	Sender  node.NodeInfo
	Message proto.Message
}

// Network is used to exchange messages with other nodes participating in the
// network.
type Network interface {
	// Listen starts listening on the given address, does not block.
	Listen() error

	// Dial returns an already connected Peer or if the connection does not
	// exist attempts to establish it.
	Dial(node node.NodeInfo) (Peer, error)

	// FindActive returns an already connected Peer.
	FindActive(id node.ID) (Peer, error)

	// Subscribe returns a channel on which it is possible to receive all
	// incoming messages. CancelFunc must be called afterwards.
	Subscribe() (chan IncomingMessage, CancelFunc)
}

// Peer strips certain methods from peer.Peer.
type Peer interface {
	// Returns information about the peer.
	Info() node.NodeInfo

	// Returns the node's public key.
	PubKey() crypto.PublicKey

	// Sends a message to a node.
	Send(proto.Message) error

	// Sends a message to the node, returns an error if context is closed.
	SendWithContext(context.Context, proto.Message) error
}
