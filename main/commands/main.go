package commands

import "github.com/boreq/netblog/cli"

var MainCmd = cli.Command{
	Options: []cli.Option{
		cli.Option{
			Name: "version",
			Type: cli.Bool,
			Description: "Display version",
		},
	},
	Run: func(c cli.Context) error {
		if c.Options["version"].Bool() {
			return nil
		}
		return cli.ErrInvalidParms
	},
	ShortDescription: "distributed blogging platform",
	Description:
`Main command decription.
Second line.`,
}

