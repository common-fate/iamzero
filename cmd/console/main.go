package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/common-fate/iamzero/cmd/console/app"
	"github.com/common-fate/iamzero/internal/tracing"
	"github.com/common-fate/iamzero/pkg/service"
	"github.com/common-fate/iamzero/pkg/storage"
	"github.com/common-fate/iamzero/pkg/tokens"
	"github.com/peterbourgon/ff/v3/ffcli"
	"go.uber.org/zap"
)

type ConsoleCommand struct {
	TracingFactory    *tracing.TracingFactory
	TokenStoreFactory *tokens.TokensStoreFactory
	Collector         *app.Console
	Svc               *service.Service
}

func main() {
	cmd := NewConsoleCommand()

	if err := cmd.ParseAndRun(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func NewConsoleCommand() *ffcli.Command {
	c := ConsoleCommand{}

	c.TracingFactory = tracing.NewFactory()
	c.TokenStoreFactory = tokens.NewFactory()
	c.Collector = app.New()
	c.Svc = service.NewService()

	fs := flag.NewFlagSet("iamzero-console", flag.ExitOnError)

	// register CLI flags for other components
	c.TracingFactory.AddFlags(fs)
	c.TokenStoreFactory.AddFlags(fs)
	c.Collector.AddFlags(fs)
	c.Svc.AddFlags(fs)

	return &ffcli.Command{
		Name:       "iamzero-console",
		ShortUsage: "IAM Zero console serves the IAM Zero web application and API",
		ShortHelp:  "Run an IAM Zero console.",
		FlagSet:    fs,
		Exec:       c.Exec,
	}
}

func (c *ConsoleCommand) Exec(ctx context.Context, _ []string) error {
	if err := c.Svc.Start(); err != nil {
		return err
	}

	log := c.Svc.Logger
	tracer, err := c.TracingFactory.InitializeTracer(ctx)
	if err != nil {
		return err
	}
	store, err := c.TokenStoreFactory.GetTokensStore(ctx, &tokens.TokensFactorySetupOpts{Log: log, Tracer: tracer})
	if err != nil {
		return err
	}

	// TODO: shift these to be configurable factories, similar to TokenStoreFactory
	actionStorage := storage.NewAlertStorage()
	policyStorage := storage.NewPolicyStorage()

	console := c.Collector

	if err := console.Start(&app.ConsoleOptions{
		Logger:        log,
		Tracer:        tracer,
		TokenStore:    store,
		ActionStorage: actionStorage,
		PolicyStorage: policyStorage,
	}); err != nil {
		return err
	}

	c.Svc.RunAndThen(func() {
		if err := console.Close(); err != nil {
			log.Fatal("failed to close console", zap.Error(err))
		}
	})
	return nil
}
