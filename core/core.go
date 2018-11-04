package core

import (
	"github.com/boreq/starlight/config"
	"github.com/boreq/starlight/core/channel"
	"github.com/boreq/starlight/core/dht"
	"github.com/boreq/starlight/core/msgregister"
	"github.com/boreq/starlight/network/dispatcher"
	"github.com/boreq/starlight/network/node"
	"github.com/boreq/starlight/protocol/message"
	"github.com/boreq/starlight/utils"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"sync"
)

var log = utils.GetLogger("core")

func NewCore(ctx context.Context, ident node.Identity, config *config.Config, dht dht.DHT) Core {
	rv := &core{
		config:      config,
		ident:       ident,
		msgRegister: msgregister.New(),
		disp:        dispatcher.New(ctx),
		dht:         dht,
		ctx:         ctx,
	}
	go rv.listenToDht()
	return rv
}

type core struct {
	config        *config.Config
	ident         node.Identity
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

func (n *core) listenToDht() {
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
}

func (n *core) Start() error {
	err := n.dht.Init(n.config.BootstrapNodes)
	if err != nil {
		return errors.Wrap(err, "dht init failed")
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
