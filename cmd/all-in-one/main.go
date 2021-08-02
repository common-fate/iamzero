package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	collectorApp "github.com/common-fate/iamzero/cmd/collector/app"
	consoleApp "github.com/common-fate/iamzero/cmd/console/app"
	"github.com/common-fate/iamzero/internal/tracing"
	"github.com/common-fate/iamzero/pkg/service"
	"github.com/common-fate/iamzero/pkg/storage"
	"github.com/common-fate/iamzero/pkg/tokens"
	"github.com/peterbourgon/ff"
	"github.com/peterbourgon/ff/v3/ffcli"
	"go.uber.org/zap"
)

type AllInOneCommand struct {
	TracingFactory    *tracing.TracingFactory
	TokenStoreFactory *tokens.TokensStoreFactory
	Collector         *collectorApp.Collector
	Console           *consoleApp.Console
}

func main() {
	cmd := NewAllInOneCommand()

	if err := cmd.ParseAndRun(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func NewAllInOneCommand() *ffcli.Command {
	c := AllInOneCommand{}

	c.TracingFactory = tracing.NewFactory()
	c.TokenStoreFactory = tokens.NewFactory()
	c.Collector = collectorApp.New()
	c.Console = consoleApp.New()

	fs := flag.NewFlagSet("iamzero-collector", flag.ExitOnError)

	// register CLI flags for other components
	c.TracingFactory.AddFlags(fs)
	c.TokenStoreFactory.AddFlags(fs)
	c.Collector.AddFlags(fs)
	c.Console.AddFlags(fs)

	return &ffcli.Command{
		Name:       "iamzero-collector",
		ShortUsage: "IAM Zero collector receives events dispatched by IAM Zero clients",
		ShortHelp:  "Run an IAM Zero collector.",
		FlagSet:    fs,
		// allow setting environment variables to configure server settings
		Options: []ff.Option{ff.WithEnvVarPrefix("IAMZERO")},
		Exec:    c.Exec,
	}
}

func (c *AllInOneCommand) Exec(ctx context.Context, _ []string) error {
	svc := service.NewService(10866)
	if err := svc.Start(); err != nil {
		return err
	}

	log := svc.Logger
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

	if err := c.Collector.Start(&collectorApp.CollectorOptions{
		Logger:        log,
		Tracer:        tracer,
		TokenStore:    store,
		ActionStorage: actionStorage,
		PolicyStorage: policyStorage,
	}); err != nil {
		return err
	}

	if err := c.Console.Start(&consoleApp.ConsoleOptions{
		Logger:        log,
		Tracer:        tracer,
		TokenStore:    store,
		ActionStorage: actionStorage,
		PolicyStorage: policyStorage,
	}); err != nil {
		return err
	}

	svc.RunAndThen(func() {
		if err := c.Collector.Close(); err != nil {
			log.Fatal("failed to close collector", zap.Error(err))
		}
		if err := c.Console.Close(); err != nil {
			log.Fatal("failed to close console", zap.Error(err))
		}
	})
	return nil
}
