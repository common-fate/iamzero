package app

import (
	"context"
	"flag"
	"net/http"
	"time"

	"github.com/common-fate/iamzero/pkg/audit"
	"github.com/common-fate/iamzero/pkg/storage"
	"github.com/common-fate/iamzero/pkg/tokens"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type Console struct {
	log           *zap.SugaredLogger
	tracer        trace.Tracer
	tokenStore    tokens.TokenStorer
	actionStorage storage.ActionStorage
	policyStorage storage.PolicyStorage
	auditor       *audit.Auditor

	Host string

	// used to hold the server so that we can shut it down
	httpServer *http.Server
}

func New() *Console {
	return &Console{}
}

type ConsoleOptions struct {
	Logger        *zap.SugaredLogger
	Tracer        trace.Tracer
	TokenStore    tokens.TokenStorer
	ActionStorage storage.ActionStorage
	PolicyStorage storage.PolicyStorage
	Auditor       *audit.Auditor
}

func (c *Console) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.Host, "console-host", "0.0.0.0:14321", "the console hostname to listen on")
}

func (c *Console) Start(opts *ConsoleOptions) error {
	c.log = opts.Logger
	c.tracer = opts.Tracer
	c.tokenStore = opts.TokenStore
	c.actionStorage = opts.ActionStorage
	c.policyStorage = opts.PolicyStorage
	c.auditor = opts.Auditor

	c.log.With("console-host", c.Host).Info("starting IAM Zero console")

	errorLog, _ := zap.NewStdLogAt(c.log.Desugar(), zap.ErrorLevel)

	server := &http.Server{
		Addr:     c.Host,
		ErrorLog: errorLog,
		Handler:  c.GetConsoleRoutes(),
	}

	c.httpServer = server

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			if err != http.ErrServerClosed {
				c.log.Errorw("Could not start console HTTP server", zap.Error(err))
			}
		}
	}()

	return nil
}

func (c *Console) Close() error {
	if c.httpServer != nil {
		timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := c.httpServer.Shutdown(timeout); err != nil {
			c.log.With(zap.Error(err)).Fatal("failed to stop the console HTTP server")
		}
		defer cancel()
	}

	return nil
}
