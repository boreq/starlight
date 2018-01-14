package commands

import (
	"fmt"
	"github.com/boreq/starlight/cli"
	"github.com/boreq/starlight/config"
	"github.com/boreq/starlight/network/node"
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
