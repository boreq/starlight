package core

import (
	"github.com/boreq/lainnet/core/dht"
	"github.com/boreq/lainnet/network/node"
	"golang.org/x/net/context"
)

// Message received from the network.
type Message struct {
	AuthorNick string
	AuthorNode node.ID
	// Local nickname of the user who should be a recipient of the message
	// or a channel name.
	Target string
	Text   string
}

type Lainnet interface {
	// Start starts listening to incoming network connections and
	// initializes the DHT.
	Start() error

	// Identity returns the identity of the local node.
	Identity() node.Identity

	// Dht returns the used DHT instance.
	Dht() dht.DHT

	// SendMessage sends a private text message to other node.
	SendMessage(ctx context.Context, to node.ID, text string) error

	// JoinChannel joins a channel. That means that the local node, declares
	// the channel membership in the DHT and starts accepting and relying
	// the messages sent in that channel.
	JoinChannel(name string) error

	// PartChannel parts a channel.
	PartChannel(name string) error
}
