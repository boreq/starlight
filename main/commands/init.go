package commands

import (
	"github.com/boreq/netblog/cli"
	"github.com/boreq/netblog/config"
)

var initCmd = cli.Command{
	Options: []cli.Option{
	},
	Run: runInit,
	ShortDescription: "initializes configuration",
	Description:
`Creates a new config file with default configuration values and generates a new
keypair.`,
}

func runInit(c cli.Context) error {
	conf := config.Default()
	e := conf.Save()
	return e
}
