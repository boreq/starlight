package commands

import (
	"errors"
	"fmt"
	"github.com/boreq/guinea"
	"github.com/boreq/starlight/config"
	"github.com/boreq/starlight/network/node"
	"github.com/boreq/starlight/utils"
	"os"
)

const defaultKeypairBits = 4096

var initCmd = guinea.Command{
	Options: []guinea.Option{
		{
			Name:        "f",
			Type:        guinea.Bool,
			Description: "Overwrite existing config",
		},
		{
			Name:        "b",
			Type:        guinea.Int,
			Default:     defaultKeypairBits,
			Description: fmt.Sprintf("Number of bits in the generated RSA key (default %d)", defaultKeypairBits),
		},
	},
	Run:              runInit,
	ShortDescription: "initializes configuration",
	Description: `
Creates a new config file with default configuration values and generates a new
keypair.`,
}

func runInit(c guinea.Context) error {
	if !c.Options["f"].Bool() {
		_, err := os.Stat(config.GetDirPath())
		if err == nil || !os.IsNotExist(err) {
			return errors.New("config already exists, use '-f' to overwrite")
		}
	}

	// Generate default config.
	utils.EnsureDirExists(config.GetDirPath(), true)
	conf := config.Default()
	err := conf.Save(config.GetConfigPath())
	if err != nil {
		return err
	}

	// Generate new identity.
	bits := c.Options["b"].Int()
	iden, err := node.GenerateIdentity(bits)
	if err != nil {
		return err
	}
	if err := node.SaveLocalIdentity(iden, config.GetDirPath()); err != nil {
		return err
	}

	return err
}
