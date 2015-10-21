package dht

import (
	"github.com/boreq/lainnet/network/node"
	"time"
)

type DHT interface {
	// Initializes the DHT using known bootstrap nodes.
	Init([]node.NodeInfo) error

	// Ping sends a ping message to a node and returns the time which was
	// needed for the node to respond.
	Ping(node.ID) (*time.Duration, error)

	// FindNode attempts to locate a node and return its address.
	FindNode(node.ID) (node.NodeInfo, error)
}
