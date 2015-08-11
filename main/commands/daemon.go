package commands

import (
	"fmt"
	"github.com/boreq/netblog/cli"
	"github.com/boreq/netblog/config"
	"github.com/boreq/netblog/core"
	"github.com/boreq/netblog/network/node"
	"golang.org/x/net/context"
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

	ctx := context.Background()

	netblog := core.NewNetblog(ctx, *iden, conf)
	netblog.Start()

	var i int
	fmt.Scanf("%d", &i)

	return nil
}
