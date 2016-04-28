package peer

import (
	"github.com/boreq/lainnet/crypto"
	"github.com/boreq/lainnet/network/node"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
)

type Peer interface {
	// Returns information about the node.
	Info() node.NodeInfo

	// Returns the node's public key.
	PubKey() crypto.PublicKey

	// Sends a message to the node.
	Send(proto.Message) error

	// Sends a message to the node, returns an error if context is closed.
	SendWithContext(context.Context, proto.Message) error

	// Receives a message from the node.
	Receive() (proto.Message, error)

	// Receives a message from the node, returns an error if context is
	// closed.
	ReceiveWithContext(context.Context) (proto.Message, error)

	// Close ends communication with the node, closes the underlying
	// connection.
	Close()

	// Closed returns true if this peer has been closed.
	Closed() bool
}
