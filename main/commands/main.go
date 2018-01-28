package commands

import "github.com/boreq/starlight/cli"

var MainCmd = cli.Command{
	Options: []cli.Option{
		cli.Option{
			Name:        "version",
			Type:        cli.Bool,
			Description: "Display version",
		},
	},
	Run: func(c cli.Context) error {
		if c.Options["version"].Bool() {
			return nil
		}
		return cli.ErrInvalidParms
	},
	Subcommands: map[string]*cli.Command{
		"daemon":   &daemonCmd,
		"init":     &initCmd,
		"identity": &identityCmd,
		"ping":     &pingCmd,
	},
	ShortDescription: "distributed chat network",
	Description: `Starlight is a distributed chat network inspired by the functionality of the
Internet Relay Chat.`,
}
