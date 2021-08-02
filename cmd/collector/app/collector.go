package app

import (
	"context"
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
	actionStorage storage.ActionStorage
	policyStorage storage.PolicyStorage

	// used to hold the server so that we can shut it down
	httpServer *http.Server
}

type CollectorParams struct {
	Logger        *zap.SugaredLogger
	Tracer        trace.Tracer
	TokenStore    tokens.TokenStorer
	Demo          bool
	ActionStorage storage.ActionStorage
	PolicyStorage storage.PolicyStorage
}

func New(params *CollectorParams) *Collector {
	return &Collector{
		log:           params.Logger,
		tracer:        params.Tracer,
		tokenStore:    params.TokenStore,
		demo:          params.Demo,
		actionStorage: params.ActionStorage,
		policyStorage: params.PolicyStorage,
	}
}

type CollectorOptions struct {
	CollectorHTTPHostPort string
}

func (c *Collector) Start(opts *CollectorOptions) error {
	c.log.With("host", opts.CollectorHTTPHostPort).Info("starting IAM Zero collector server")

	errorLog, _ := zap.NewStdLogAt(c.log.Desugar(), zap.ErrorLevel)

	server := &http.Server{
		Addr:     opts.CollectorHTTPHostPort,
		ErrorLog: errorLog,
	}

	server.Handler = c.GetCollectorRoutes()

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
			c.log.Fatal("failed to stop the collector HTTP server", "error", err)
		}
		defer cancel()
	}

	return nil
}
