package network

import (
	"github.com/boreq/netblog/network/node"
	"github.com/boreq/netblog/protocol"
)

// Since all incoming messages are passed on the same channel they must be
// bundled with a node.ID.
type IncomingMessage struct {
	node.NodeInfo
	protocol.Message
}

// Network is used to exchange messages with other nodes participating in the
// network.
type Network interface {
	// Listen starts listening on the given address, does not block.
	Listen(address string) error

	// Dial returns Peer which allows to send messages to other nodes.
	Dial(node node.NodeInfo) (Peer, error)

	// Subscribe returns a channel on which it is possible to receive all
	// incoming messages. CancelFunc must be called afterwards.
	Subscribe() (chan IncomingMessage, CancelFunc)
}

type Peer interface {
	// Returns information about the peer.
	Info() node.NodeInfo

	// Sends a message to a node.
	Send(protocol.Message) error
}
