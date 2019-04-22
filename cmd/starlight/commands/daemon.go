package commands

import (
	"github.com/boreq/guinea"
	"github.com/boreq/starlight/core"
	"github.com/boreq/starlight/core/dht"
	"github.com/boreq/starlight/irc"
	"github.com/boreq/starlight/local"
	"github.com/boreq/starlight/network"
	"golang.org/x/net/context"
	"os"
)

var daemonCmd = guinea.Command{
	Run:              daemon,
	ShortDescription: "runs a daemon",
}

func daemon(c guinea.Context) error {
	conf, err := GetConfig()
	if err != nil {
		return err
	}

	iden, err := GetIdentity()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to the wired
	net := network.New(ctx, *iden, conf.ListenAddress)
	dht := dht.New(ctx, net, *iden)
	core := core.NewCore(ctx, *iden, conf, dht)

	err = net.Listen()
	if err != nil {
		return err
	}

	err = core.Start()
	if err != nil {
		return err
	}

	// Run the local API server
	address := local.GetAddress(iden.Id)
	err = os.Remove(address)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	defer os.Remove(address)
	err = local.RunServer(core, address)
	if err != nil {
		return err
	}

	// Run the local IRC gateway
	ircSrv := irc.NewServer(core)
	err = ircSrv.Start(ctx, conf.IRCGatewayAddress)
	if err != nil {
		return err
	}

	<-ctx.Done()

	return nil
}
