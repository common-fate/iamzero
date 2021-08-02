package app

import (
	"context"
	"flag"
	"net/http"
	"time"

	"github.com/common-fate/iamzero/pkg/storage"
	"github.com/common-fate/iamzero/pkg/tokens"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type Collector struct {
	log           *zap.SugaredLogger
	tracer        trace.Tracer
	tokenStore    tokens.TokenStorer
	demo          bool
	actionStorage *storage.ActionStorage
	policyStorage *storage.PolicyStorage

	Host string
	Demo bool

	// used to hold the server so that we can shut it down
	httpServer *http.Server
}

func New() *Collector {
	return &Collector{}
}

type CollectorOptions struct {
	Logger        *zap.SugaredLogger
	Tracer        trace.Tracer
	TokenStore    tokens.TokenStorer
	ActionStorage *storage.ActionStorage
	PolicyStorage *storage.PolicyStorage
}

func (c *Collector) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.Host, "collector-host", "0.0.0.0:13991", "the collector hostname to listen on")
	fs.BoolVar(&c.Demo, "collector-demo", false, "run in demo mode, censoring AWS role info")
}

func (c *Collector) Start(opts *CollectorOptions) error {
	c.log = opts.Logger
	c.tracer = opts.Tracer
	c.tokenStore = opts.TokenStore
	c.actionStorage = opts.ActionStorage
	c.policyStorage = opts.PolicyStorage

	c.log.With("collector-host", c.Host).Info("starting IAM Zero collector server")

	errorLog, _ := zap.NewStdLogAt(c.log.Desugar(), zap.ErrorLevel)

	server := &http.Server{
		Addr:     c.Host,
		ErrorLog: errorLog,
		Handler:  c.GetCollectorRoutes(),
	}

	c.httpServer = server

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			if err != http.ErrServerClosed {
				c.log.Errorw("Could not start HTTP collector", zap.Error(err))
			}
		}
	}()

	return nil
}

func (c *Collector) Close() error {
	if c.httpServer != nil {
		timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := c.httpServer.Shutdown(timeout); err != nil {
			c.log.With(zap.Error(err)).Fatal("failed to stop the collector HTTP server")
		}
		defer cancel()
	}

	return nil
}
