package commands

import (
	"fmt"
	"github.com/boreq/guinea"
)

var identityCmd = guinea.Command{
	Run:              runIdentity,
	ShortDescription: "displays local identity",
	Description: `
Displays your identity.`,
}

func runIdentity(c guinea.Context) error {
	iden, err := GetIdentity()
	if err != nil {
		return err
	}
	fmt.Println(iden.Id)
	return nil
}
