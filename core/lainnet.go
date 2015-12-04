package core

import (
	"github.com/boreq/lainnet/config"
	"github.com/boreq/lainnet/core/channel"
	"github.com/boreq/lainnet/core/dht"
	"github.com/boreq/lainnet/network"
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
		config: config,
		ident:  ident,
		net:    net,
		dht:    dht.New(ctx, net, ident),
		ctx:    ctx,
	}
	return rv
}

type lainnet struct {
	config        *config.Config
	ident         node.Identity
	net           network.Network
	channels      []*channel.Channel
	channelsMutex sync.Mutex
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
		c, cancel := n.net.Subscribe()
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

func (n *lainnet) SendMessage(ctx context.Context, id node.ID, text string) error {
	p, err := n.dht.Dial(ctx, id)
	if err != nil {
		return err
	}
	msg := &message.PrivateMessage{Text: &text}
	return p.Send(msg)
}

// handleMessage handles the incoming messages.
func (n *lainnet) handleMessage(msg network.IncomingMessage) error {
	switch pMsg := msg.Message.(type) {
	case *message.PrivateMessage:
		log.Print(pMsg)
	}
	return nil
}
