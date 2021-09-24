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

type Collector struct {
	log           *zap.SugaredLogger
	tracer        trace.Tracer
	tokenStore    tokens.TokenStorer
	demo          bool
	actionStorage storage.ActionStorage
	policyStorage storage.PolicyStorage
	auditor       *audit.Auditor

	// whether to enable the AWS CDK resource integration
	CDK                   bool
	TF                    bool
	Host                  string
	Demo                  bool
	TransportSQSEnabled   bool
	TransportSQSQueueURL  string
	TransportSQSTokenAuth bool

	// used to hold the server so that we can shut it down
	httpServer *http.Server
	sqsServer  *SQSServer
}

func New() *Collector {
	return &Collector{}
}

type CollectorOptions struct {
	Logger        *zap.SugaredLogger
	Tracer        trace.Tracer
	Auditor       *audit.Auditor
	TokenStore    tokens.TokenStorer
	ActionStorage storage.ActionStorage
	PolicyStorage storage.PolicyStorage
}

func (c *Collector) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.Host, "collector-host", "0.0.0.0:13991", "the collector hostname to listen on")
	fs.BoolVar(&c.Demo, "collector-demo", false, "run in demo mode, censoring AWS role info")
	fs.BoolVar(&c.CDK, "cdk", false, "whether to audit CDK resources (requires at least one audit-role argument to be passed)")
	fs.BoolVar(&c.TF, "tf", false, "whether to audit Terraform resources (requires at least one audit-role argument to be passed)")
	fs.BoolVar(&c.TransportSQSEnabled, "transport-sqs-enabled", false, "enable SQS collector transport")
	fs.BoolVar(&c.TransportSQSTokenAuth, "transport-sqs-token-auth", true, "verify IAM Zero token on events received via SQS")
	fs.StringVar(&c.TransportSQSQueueURL, "transport-sqs-queue-url", "", "(if SQS transport enabled) the SQS queue URL")
}

func (c *Collector) Start(ctx context.Context, opts *CollectorOptions) error {
	c.log = opts.Logger
	c.tracer = opts.Tracer
	c.auditor = opts.Auditor
	c.tokenStore = opts.TokenStore
	c.actionStorage = opts.ActionStorage
	c.policyStorage = opts.PolicyStorage

	c.auditor.Setup(c.log)

	// err := c.auditor.LoadResources(ctx)
	// if err != nil {
	// 	return err
	// }

	if c.CDK {
		err := c.auditor.LoadCloudFormationStacks(ctx)
		if err != nil {
			return err
		}
	}

	c.auditor.LoadFixture()

	c.log.With("roles", c.auditor.GetRoles()).Info("auditor: found roles")

	c.auditor.BuildLinks()

	c.log.With("links", c.auditor.GetLinks()).Info("auditor: found links")

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

	if c.TransportSQSEnabled {
		ctx := context.Background()

		server, err := NewSQSServer(ctx, &SQSServerConfig{
			Log:      c.log,
			Tracer:   c.tracer,
			QueueUrl: c.TransportSQSQueueURL,
			Handler:  c.HandleSQSMessage,
		})
		if err != nil {
			return err
		}
		c.sqsServer = server

		c.log.With("queue-url", server.QueueUrl()).Info("starting SQS transport listener")

		server.Start(ctx)
	}

	return nil
}

func (c *Collector) Close() error {
	if c.httpServer != nil {
		timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := c.httpServer.Shutdown(timeout); err != nil {
			c.log.With(zap.Error(err)).Fatal("failed to stop the collector HTTP server")
		}
		if c.sqsServer != nil {
			c.sqsServer.Shutdown()
		}
		defer cancel()
	}

	return nil
}
