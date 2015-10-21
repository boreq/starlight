package commands

import (
	"fmt"
	"github.com/boreq/lainnet/cli"
	"github.com/boreq/lainnet/config"
	"github.com/boreq/lainnet/network/node"
)

var identityCmd = cli.Command{
	Run:              runIdentity,
	ShortDescription: "displays local identity",
	Description: `
Displays your identity.`,
}

func runIdentity(c cli.Context) error {
	iden, err := node.LoadLocalIdentity(config.GetDir())
	if err != nil {
		return err
	}
	fmt.Println(iden.Id)
	return nil
}
