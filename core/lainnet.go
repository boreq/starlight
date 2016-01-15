package core

import (
	"github.com/boreq/lainnet/config"
	"github.com/boreq/lainnet/core/channel"
	"github.com/boreq/lainnet/core/dht"
	"github.com/boreq/lainnet/core/msgregister"
	"github.com/boreq/lainnet/network"
	"github.com/boreq/lainnet/network/dispatcher"
	"github.com/boreq/lainnet/network/node"
	"github.com/boreq/lainnet/protocol/message"
	"github.com/boreq/lainnet/utils"
	"golang.org/x/net/context"
	"sync"
)

var log = utils.GetLogger("lainnet")

func NewLainnet(ctx context.Context, ident node.Identity, config *config.Config) Lainnet {
	net := network.New(ctx, ident, config.ListenAddress)
	rv := &lainnet{
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

type lainnet struct {
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

func (n *lainnet) Identity() node.Identity {
	return n.ident
}

func (n *lainnet) Dht() dht.DHT {
	return n.dht
}

func (n *lainnet) Start() error {
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

func (n *lainnet) Subscribe() (chan dispatcher.IncomingMessage, dispatcher.CancelFunc) {
	return n.disp.Subscribe()
}

// handleMessage handles the incoming messages.
func (n *lainnet) handleMessage(msg dispatcher.IncomingMessage) error {
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
