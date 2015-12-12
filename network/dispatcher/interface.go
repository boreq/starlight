// Dispatcher can dispatch incoming messages to multiple receivers.
package dispatcher

import (
	"github.com/boreq/lainnet/network/node"
	"github.com/golang/protobuf/proto"
)

type CancelFunc func()

// Dispatcher exposes methods which allow to subscribe to messages. Dispatched
// messages are sent to every subscriber through a channel.
type Dispatcher interface {
	// Subscribe returns a channel on which it is possible to receive
	// incoming messages and a CancelFunc which must be called if the
	// calling function no longer wishes to receive messages through a
	// returned channel.
	Subscribe() (chan IncomingMessage, CancelFunc)

	// Dispatch forwards a message to all channels retrieved using the
	// subscribe method.
	Dispatch(node.NodeInfo, proto.Message)
}

// Since all incoming messages are passed on the same channel they must be
// bundled with information about the sender.
type IncomingMessage struct {
	Sender  node.NodeInfo
	Message proto.Message
}
