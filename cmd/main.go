package main

import (
	"context"
	"fmt"
	"os"

	"github.com/common-fate/iamzero/cmd/commands"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func main() {
	var (
		out                     = os.Stdout
		rootCommand, rootConfig = commands.RootCommand()
		localCommand            = commands.NewLocalCommand(rootConfig, out)
		serverCommand           = commands.NewServerCommand(rootConfig, out)
	)

	rootCommand.Subcommands = []*ffcli.Command{
		localCommand,
		serverCommand,
	}

	if err := rootCommand.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error during Parse: %v\n", err)
		os.Exit(1)
	}

	if err := rootCommand.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
