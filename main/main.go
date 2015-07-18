package main

import (
	"os"
	"strings"
	"github.com/boreq/netblog/cli"
	"github.com/boreq/netblog/main/commands"
)

var globalOpt = []cli.Option{
	cli.Option{
		Name: "help",
		Type: cli.Bool,
		Default: false,
		Description: "Display help",
	},
}

func findCommand(cmd *cli.Command, args []string) (*cli.Command, []string) {
	for name, subCmd := range cmd.Subcommands {
		if len(args) > 0 && args[0] == name {
			return findCommand(subCmd, args[1:])
		}
	}
	return cmd, args
}

func main() {
	c, args := findCommand(&commands.MainCmd, os.Args[1:])
	argOffset := len(os.Args) - len(args)
	foundCmdName := strings.Join(os.Args[:argOffset], " ")
	c.Options = append(c.Options, globalOpt...)
	e := c.Execute(foundCmdName, globalOpt, args)
	if e != nil {
		panic(e)
	}
}
