package commands

import (
	"github.com/boreq/starlight/cli"
	"github.com/boreq/starlight/config"
	"github.com/boreq/starlight/core"
	"github.com/boreq/starlight/irc"
	"github.com/boreq/starlight/local"
	"github.com/boreq/starlight/network/node"
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
	core := core.NewCore(ctx, *iden, conf)
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
