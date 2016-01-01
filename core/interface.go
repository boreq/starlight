package core

import (
	"github.com/boreq/lainnet/core/dht"
	"github.com/boreq/lainnet/network/dispatcher"
	"github.com/boreq/lainnet/network/node"
	"golang.org/x/net/context"
)

type Lainnet interface {
	// Start starts listening to incoming network connections and
	// initializes the DHT.
	Start() error

	// Identity returns the identity of the local node.
	Identity() node.Identity

	// Dht returns the used DHT instance.
	Dht() dht.DHT

	// Subscribe returns a channel on which it is possible to receive
	// incoming PrivateMessage and ChannelMessage messages. CancelFunc must
	// be called afterwards. ChannelMessage will have a ChannelId replaced
	// with a string containing the actual name of the channel which was
	// provided to the the JoinChannel method.
	Subscribe() (chan dispatcher.IncomingMessage, dispatcher.CancelFunc)

	// SendMessage sends a private text message to a node.
	SendMessage(ctx context.Context, to node.ID, text string) error

	// SendChannelMessage sends a text message to a specified channel.
	SendChannelMessage(ctx context.Context, channel string, text string) error

	// JoinChannel joins a channel. That means that the local node, declares
	// the channel membership in the DHT and starts accepting and relying
	// the messages sent in that channel.
	JoinChannel(name string) error

	// PartChannel parts a channel.
	PartChannel(name string) error
}
