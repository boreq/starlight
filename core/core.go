package core

import (
	"github.com/boreq/lainnet/config"
	"github.com/boreq/lainnet/core/dht"
	"github.com/boreq/lainnet/network/node"
	"golang.org/x/net/context"
)

type Netblog interface {
	Start() error
	Identity() node.Identity
	Dht() dht.DHT
}

func NewNetblog(ctx context.Context, ident node.Identity, config *config.Config) Netblog {
	rw := &netblog{
		config: config,
		ident:  ident,
		dht:    dht.New(ctx, ident, config.ListenAddress),
		ctx:    ctx,
	}
	return rw
}

type netblog struct {
	config *config.Config
	ident  node.Identity
	dht    dht.DHT
	ctx    context.Context
}

func (n *netblog) Identity() node.Identity {
	return n.ident
}

func (n *netblog) Dht() dht.DHT {
	return n.dht
}

func (n *netblog) Start() error {
	err := n.dht.Init(n.config.BootstrapNodes)
	if err != nil {
		return err
	}

	return nil
}
