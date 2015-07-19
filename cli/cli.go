package cli

import (
	"errors"
	"flag"
	"fmt"
	"strings"
)

var ErrInvalidParms = errors.New("invalid parameters")

type ValType int

const (
	String ValType = iota
	Bool
	Int
)

// Used to generate flags use by the flag module.
type Option struct {
	Name        string
	Type        ValType
	Default     interface{}
	Description string
}

func (opt Option) String() string {
	prefix := "-"
	if len(opt.Name) > 1 {
		prefix = "--"
	}
	return prefix + opt.Name
}

// Used only for generating help.
type Argument struct {
	Name        string
	Multiple    bool
	Description string
}

func (arg Argument) String() string {
	format := "<%s>"
	if arg.Multiple {
		format += "..."
	}
	return fmt.Sprintf(format, arg.Name)
}

type OptionValue struct {
	Value interface{}
}

func (v OptionValue) Bool() bool {
	return *v.Value.(*bool)
}

func (v OptionValue) Int() int {
	return *v.Value.(*int)
}

type Context struct {
	Options   map[string]OptionValue
	Arguments []string
}

func makeContext(c Command, args []string) (*Context, error) {
	context := &Context{
		Options: make(map[string]OptionValue),
	}

	flagset := flag.NewFlagSet("sth", flag.ContinueOnError)
	flagset.Usage = func() {}
	for _, option := range c.Options {
		switch option.Type {
		case String:
			if option.Default == nil {
				option.Default = ""
			}
			context.Options[option.Name] = OptionValue{
				Value: flagset.String(option.Name, option.Default.(string), ""),
			}
		case Bool:
			if option.Default == nil {
				option.Default = false
			}
			context.Options[option.Name] = OptionValue{
				Value: flagset.Bool(option.Name, option.Default.(bool), ""),
			}
		case Int:
			if option.Default == nil {
				option.Default = 0
			}
			context.Options[option.Name] = OptionValue{
				Value: flagset.Int(option.Name, option.Default.(int), ""),
			}
		}
	}
	e := flagset.Parse(args)
	if e != nil {
		return nil, e
	}
	context.Arguments = flagset.Args()
	return context, nil
}

type CommandFunction func(Context) error

type Command struct {
	Options          []Option
	Run              CommandFunction
	Subcommands      map[string]*Command
	Arguments        []Argument
	ShortDescription string
	Description      string
}

func (c Command) UsageString(cmdName string) string {
	rw := cmdName
	if len(c.Subcommands) > 0 {
		rw += " <subcommand>"
	}
	rw += " [<options>]"
	for _, arg := range c.Arguments {
		rw += fmt.Sprintf(" %s", arg)
	}
	return rw
}

func (c Command) PrintHelp(cmdName string) {
	usage := c.UsageString(cmdName)
	fmt.Printf("\n    %s - %s\n", usage, c.ShortDescription)

	if len(c.Options) > 0 {
		fmt.Println("\nOPTIONS:")
		for _, opt := range c.Options {
			fmt.Printf("    %-20s %s\n", opt, opt.Description)
		}
	}

	if len(c.Arguments) > 0 {
		fmt.Println("\nARGUMENTS:")
		for _, arg := range c.Arguments {
			fmt.Printf("    %-20s %s\n", arg, arg.Description)
		}
	}

	if len(c.Subcommands) > 0 {
		fmt.Println("\nSUBCOMMANDS:")
		for name, subCmd := range c.Subcommands {
			fmt.Printf("    %-20s %s\n", name, subCmd.ShortDescription)
		}
		fmt.Printf("\n    Try '%s <subcommand> --help'\n", cmdName)
	}

	if len(c.Description) > 0 {
		fmt.Println("\nDESCRIPTION:")
		desc := strings.Trim(c.Description, "\n")
		for _, line := range strings.Split(desc, "\n") {
			fmt.Printf("    %s\n", line)
		}
	}
}

func (c Command) Execute(cmdName string, globalOpt []Option, args []string) error {
	context, e := makeContext(c, args)
	if e != nil {
		c.PrintHelp(cmdName)
		return e
	}
	if context.Options["help"].Bool() || c.Run == nil {
		c.PrintHelp(cmdName)
		return nil
	} else {
		e := c.Run(*context)
		if e == ErrInvalidParms {
			c.PrintHelp(cmdName)
		}
		return e
	}
}
