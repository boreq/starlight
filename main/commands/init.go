package commands

import (
	"errors"
	"github.com/boreq/netblog/cli"
	"github.com/boreq/netblog/config"
	"github.com/boreq/netblog/network/node"
	"github.com/boreq/netblog/utils"
	"os"
	"path"
)

var initCmd = cli.Command{
	Options: []cli.Option{
		cli.Option{
			Name:        "f",
			Type:        cli.Bool,
			Description: "Overwrite existing config",
		},
		cli.Option{
			Name:        "b",
			Type:        cli.Int,
			Default:     4096,
			Description: "Number of bits which will be used during RSA key generation (default 4096)",
		},
	},
	Run:              runInit,
	ShortDescription: "initializes configuration",
	Description: `
Creates a new config file with default configuration values and generates a new
keypair.`,
}

func runInit(c cli.Context) error {
	if !c.Options["f"].Bool() {
		_, err := os.Stat(config.GetDir())
		if err == nil || !os.IsNotExist(err) {
			return errors.New("Config already exists. Use '-f' to overwrite.")
		}
	}

	// Generate default config.
	utils.EnsureDirExists(config.GetDir(), true)
	conf := config.Default()
	err := conf.Save()
	if err != nil {
		return err
	}

	// Generate new identity.
	// TODO: difficulty
	bits := c.Options["b"].Int()
	iden, err := node.GenerateIdentity(bits, 0)
	if err != nil {
		return err
	}
	identityDir := path.Join(config.GetDir(), "identity")
	if err := node.SaveLocalIdentity(iden, identityDir); err != nil {
		return err
	}

	return err
}
