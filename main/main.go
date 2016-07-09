package main

import (
	"fmt"
	"github.com/boreq/lainnet/cli"
	"github.com/boreq/lainnet/main/commands"
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
