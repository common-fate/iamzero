package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/common-fate/iamzero/cmd/collector/app"
	"github.com/common-fate/iamzero/internal/tracing"
	"github.com/common-fate/iamzero/pkg/service"
	"github.com/common-fate/iamzero/pkg/storage"
	"github.com/common-fate/iamzero/pkg/tokens"
	"github.com/peterbourgon/ff/v3/ffcli"
	"go.uber.org/zap"
)

type CollectorCommand struct {
	TracingFactory    *tracing.TracingFactory
	TokenStoreFactory *tokens.TokensStoreFactory
	Collector         *app.Collector
	Svc               *service.Service
}

func main() {
	cmd := NewCollectorCommand()

	if err := cmd.ParseAndRun(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func NewCollectorCommand() *ffcli.Command {
	c := CollectorCommand{}

	c.TracingFactory = tracing.NewFactory()
	c.TokenStoreFactory = tokens.NewFactory()
	c.Collector = app.New()
	c.Svc = service.NewService()

	fs := flag.NewFlagSet("iamzero-collector", flag.ExitOnError)

	// register CLI flags for other components
	c.TracingFactory.AddFlags(fs)
	c.TokenStoreFactory.AddFlags(fs)
	c.Collector.AddFlags(fs)
	c.Svc.AddFlags(fs)

	return &ffcli.Command{
		Name:       "iamzero-collector",
		ShortUsage: "IAM Zero collector receives events dispatched by IAM Zero clients",
		ShortHelp:  "Run an IAM Zero collector.",
		FlagSet:    fs,
		Exec:       c.Exec,
	}
}

func (c *CollectorCommand) Exec(ctx context.Context, _ []string) error {
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

	co := c.Collector

	if err := co.Start(ctx, &app.CollectorOptions{
		Logger:        log,
		Tracer:        tracer,
		TokenStore:    store,
		ActionStorage: actionStorage,
		PolicyStorage: policyStorage,
	}); err != nil {
		return err
	}

	c.Svc.RunAndThen(func() {
		if err := co.Close(); err != nil {
			log.Fatal("failed to close collector", zap.Error(err))
		}
	})
	return nil
}
