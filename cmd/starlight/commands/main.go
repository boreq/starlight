package commands

import (
	"github.com/boreq/guinea"
)

var MainCmd = guinea.Command{
	Options: []guinea.Option{
		{
			Name:        "version",
			Type:        guinea.Bool,
			Description: "Display version",
		},
	},
	Run: func(c guinea.Context) error {
		if c.Options["version"].Bool() {
			return nil
		}
		return guinea.ErrInvalidParms
	},
	Subcommands: map[string]*guinea.Command{
		"daemon":   &daemonCmd,
		"init":     &initCmd,
		"identity": &identityCmd,
		"ping":     &pingCmd,
	},
	ShortDescription: "distributed chat network",
	Description: `Starlight is a distributed chat network inspired by the functionality of the
Internet Relay Chat.`,
}
