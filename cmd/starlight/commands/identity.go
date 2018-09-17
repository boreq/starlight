package commands

import (
	"fmt"
	"github.com/boreq/guinea"
	"github.com/boreq/starlight/config"
	"github.com/boreq/starlight/network/node"
)

var identityCmd = guinea.Command{
	Run:              runIdentity,
	ShortDescription: "displays local identity",
	Description: `
Displays your identity.`,
}

func runIdentity(c guinea.Context) error {
	iden, err := node.LoadLocalIdentity(config.GetDir())
	if err != nil {
		return err
	}
	fmt.Println(iden.Id)
	return nil
}
