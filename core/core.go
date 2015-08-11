package core

import (
	"github.com/boreq/netblog/config"
	"github.com/boreq/netblog/core/dht"
	"github.com/boreq/netblog/network/node"
	"golang.org/x/net/context"
)

type Netblog interface {
	Start()
}

func NewNetblog(ctx context.Context, ident node.Identity, config *config.Config) Netblog {
	rw := &netblog{
		config: config,
		ident:  ident,
		dht:    dht.New(ctx, ident),
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

func (n *netblog) Start() {
	n.dht.Init(n.config.BootstrapNodes, n.config.ListenAddress)
}
