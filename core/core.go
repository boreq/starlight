package core

import (
	"github.com/boreq/starlight/config"
	"github.com/boreq/starlight/core/channel"
	"github.com/boreq/starlight/core/dht"
	"github.com/boreq/starlight/core/msgregister"
	"github.com/boreq/starlight/network"
	"github.com/boreq/starlight/network/dispatcher"
	"github.com/boreq/starlight/network/node"
	"github.com/boreq/starlight/protocol/message"
	"github.com/boreq/starlight/utils"
	"golang.org/x/net/context"
	"sync"
)

var log = utils.GetLogger("core")

func NewCore(ctx context.Context, ident node.Identity, config *config.Config) Core {
	net := network.New(ctx, ident, config.ListenAddress)
	rv := &core{
		config:      config,
		ident:       ident,
		net:         net,
		msgRegister: msgregister.New(),
		disp:        dispatcher.New(ctx),
		dht:         dht.New(ctx, net, ident),
		ctx:         ctx,
	}
	return rv
}

type core struct {
	config        *config.Config
	ident         node.Identity
	net           network.Network
	channels      []*channel.Channel
	channelsMutex sync.Mutex
	msgRegister   *msgregister.Register
	disp          dispatcher.Dispatcher
	dht           dht.DHT
	ctx           context.Context
}

func (n *core) Identity() node.Identity {
	return n.ident
}

func (n *core) Dht() dht.DHT {
	return n.dht
}

func (n *core) Start() error {
	go func() {
		c, cancel := n.dht.Subscribe()
		defer cancel()
		for {
			select {
			case msg := <-c:
				go n.handleMessage(msg)
			case <-n.ctx.Done():
				return
			}
		}
	}()

	err := n.net.Listen()
	if err != nil {
		return err
	}

	err = n.dht.Init(n.config.BootstrapNodes)
	if err != nil {
		return err
	}

	return nil
}

func (n *core) Subscribe() (chan dispatcher.IncomingMessage, dispatcher.CancelFunc) {
	return n.disp.Subscribe()
}

// handleMessage handles the incoming messages.
func (n *core) handleMessage(msg dispatcher.IncomingMessage) error {
	switch pMsg := msg.Message.(type) {

	case *message.PrivateMessage:
		n.handlePrivateMessageMsg(pMsg, msg.Sender)

	case *message.ChannelMessage:
		n.handleChannelMessageMsg(pMsg, msg.Sender)

	case *message.FindChannel:
		n.handleFindChannelMsg(pMsg, msg.Sender.Id)

	case *message.StoreChannel:
		n.handleStoreChannelMsg(pMsg, msg.Sender.Id)
	}
	return nil
}
