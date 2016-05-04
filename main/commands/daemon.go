package commands

import (
	"github.com/boreq/lainnet/cli"
	"github.com/boreq/lainnet/config"
	"github.com/boreq/lainnet/core"
	"github.com/boreq/lainnet/irc"
	"github.com/boreq/lainnet/local"
	"github.com/boreq/lainnet/network/node"
	"golang.org/x/net/context"
	"os"
)

var daemonCmd = cli.Command{
	Run:              daemon,
	ShortDescription: "runs a daemon",
}

func daemon(c cli.Context) error {
	conf, err := config.Get()
	if err != nil {
		return err
	}

	iden, err := node.LoadLocalIdentity(config.GetDir())
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to the wired
	lainnet := core.NewLainnet(ctx, *iden, conf)
	err = lainnet.Start()
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
	err = local.RunServer(lainnet, address)
	if err != nil {
		return err
	}

	// Run the local IRC gateway
	ircSrv := irc.NewServer(lainnet)
	err = ircSrv.Start(ctx, conf.IRCGatewayAddress)
	if err != nil {
		return err
	}

	<-ctx.Done()

	return nil
}
