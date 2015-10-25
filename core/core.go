package core

import (
	"github.com/boreq/lainnet/config"
	"github.com/boreq/lainnet/core/dht"
	"github.com/boreq/lainnet/network/node"
	"golang.org/x/net/context"
)

type Lainnet interface {
	Start() error
	Identity() node.Identity
	Dht() dht.DHT
}

func NewLainnet(ctx context.Context, ident node.Identity, config *config.Config) Lainnet {
	rw := &lainnet{
		config: config,
		ident:  ident,
		dht:    dht.New(ctx, ident, config.ListenAddress),
		ctx:    ctx,
	}
	return rw
}

type lainnet struct {
	config *config.Config
	ident  node.Identity
	dht    dht.DHT
	ctx    context.Context
}

func (n *lainnet) Identity() node.Identity {
	return n.ident
}

func (n *lainnet) Dht() dht.DHT {
	return n.dht
}

func (n *lainnet) Start() error {
	err := n.dht.Init(n.config.BootstrapNodes)
	if err != nil {
		return err
	}
	return nil
}
