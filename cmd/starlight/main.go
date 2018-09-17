package main

import (
	"fmt"
	"github.com/boreq/guinea"
	"github.com/boreq/starlight/cmd/starlight/commands"
	"os"
)

var globalOpt = []guinea.Option{
	guinea.Option{
		Name:        "help",
		Type:        guinea.Bool,
		Default:     false,
		Description: "Display help",
	},
}

func main() {
	cmd, cmdName, cmdArgs := guinea.FindCommand(&commands.MainCmd, os.Args)
	cmd.Options = append(cmd.Options, globalOpt...)
	e := cmd.Execute(cmdName, cmdArgs)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
	}
}
