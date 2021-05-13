package commands

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
)

// Config for the root command, including flags and types that should be
// available to each subcommand.
type RootConfig struct {
	Verbose bool
}

// New constructs a usable ffcli.Command and an empty Config. The config's token
// and verbose fields will be set after a successful parse. The caller must
// initialize the config's object API client field.
func RootCommand() (*ffcli.Command, *RootConfig) {
	var cfg RootConfig

	fs := flag.NewFlagSet("iamzero", flag.ExitOnError)
	cfg.RegisterFlags(fs)

	return &ffcli.Command{
		Name:       "iamzero",
		ShortUsage: "iamzero [flags] <subcommand> [flags] [<arg>...]",
		FlagSet:    fs,
		Options:    []ff.Option{ff.WithEnvVarPrefix("IAMZERO")},
		Exec:       cfg.Exec,
	}, &cfg
}

// RegisterFlags registers the flag fields into the provided flag.FlagSet. This
// helper function allows subcommands to register the root flags into their
// flagsets, creating "global" flags that can be passed after any subcommand at
// the commandline.
func (c *RootConfig) RegisterFlags(fs *flag.FlagSet) {
	fs.BoolVar(&c.Verbose, "v", false, "log verbose output")
}

// Exec function for this command.
func (c *RootConfig) Exec(context.Context, []string) error {
	// The root command has no meaning, so if it gets executed,
	// display the usage text to the user instead.
	return flag.ErrHelp
}
