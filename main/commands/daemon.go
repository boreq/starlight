package commands

import (
	"github.com/boreq/netblog/cli"
	"github.com/boreq/netblog/config"
	"github.com/boreq/netblog/core"
	"github.com/boreq/netblog/local"
	"github.com/boreq/netblog/network/node"
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

	netblog := core.NewNetblog(ctx, *iden, conf)
	netblog.Start()

	// Run local server.
	address := local.GetAddress(iden.Id)
	err = os.Remove(address)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	defer os.Remove(address)

	err = local.RunServer(netblog, address)
	if err != nil {
		return err
	}

	<-ctx.Done()

	return nil
}
