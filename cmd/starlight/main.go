package main

import (
	"fmt"
	"github.com/boreq/starlight/cli"
	"github.com/boreq/starlight/cmd/starlight/commands"
	"os"
)

var globalOpt = []cli.Option{
	cli.Option{
		Name:        "help",
		Type:        cli.Bool,
		Default:     false,
		Description: "Display help",
	},
}

func main() {
	cmd, cmdName, cmdArgs := cli.FindCommand(&commands.MainCmd, os.Args)
	cmd.Options = append(cmd.Options, globalOpt...)
	e := cmd.Execute(cmdName, cmdArgs)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
	}
}
