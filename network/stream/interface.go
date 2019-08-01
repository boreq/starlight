package stream

import (
	"github.com/boreq/starlight/crypto"
	"github.com/boreq/starlight/network/node"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
)

type Stream interface {
	// Info returns basic information about the node.
	Info() node.NodeInfo

	// PubKey returns the node's public key.
	PubKey() crypto.PublicKey

	// Send sends a message to the node.
	Send(proto.Message) error

	// SendWithContext sends a message to the node and returns an error if
	// the context is closed.
	SendWithContext(context.Context, proto.Message) error

	// Receive receives a message from the node.
	Receive() (proto.Message, error)

	// ReceiveWithContext receives a message from the node and returns an
	// error if the context is closed.
	ReceiveWithContext(context.Context) (proto.Message, error)

	// Close ends communication with the node, closes the underlying
	// connection.
	Close()

	// Closed returns true if this stream has been closed.
	Closed() bool
}
