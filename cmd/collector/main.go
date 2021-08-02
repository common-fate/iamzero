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
	Host string

	TracingFactory    *tracing.TracingFactory
	TokenStoreFactory *tokens.TokensStoreFactory
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

	fs := flag.NewFlagSet("iamzero-collector", flag.ExitOnError)

	fs.StringVar(&c.Host, "host", "0.0.0.0:9090", "the server hostname to listen on (can be set via IAMZERO_HOST env var)")

	// register CLI flags for other components
	c.TracingFactory.AddFlags(fs)
	c.TokenStoreFactory.AddFlags(fs)

	return &ffcli.Command{
		Name:       "iamzero-collector",
		ShortUsage: "IAM Zero collector receives events dispatched by IAM Zero clients",
		ShortHelp:  "Run an IAM Zero collector.",
		FlagSet:    fs,
		Exec:       c.Exec,
	}
}

func (c *CollectorCommand) Exec(ctx context.Context, _ []string) error {
	svc := service.NewService(10000)
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

	co := app.New(&app.CollectorParams{
		Logger:        log,
		Tracer:        tracer,
		TokenStore:    store,
		ActionStorage: actionStorage,
		PolicyStorage: policyStorage,
	})

	if err := co.Start(&app.CollectorOptions{CollectorHTTPHostPort: c.Host}); err != nil {
		return err
	}

	svc.RunAndThen(func() {
		if err := co.Close(); err != nil {
			log.Fatal("failed to close collector", zap.Error(err))
		}
	})
	return nil
}
