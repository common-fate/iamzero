package app

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/common-fate/iamzero/pkg/events"
	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/common-fate/iamzero/pkg/tokens"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// SQSServer polls an SQS queue to receive events from clients
type SQSServer struct {
	log       *zap.SugaredLogger
	tracer    trace.Tracer
	client    *sqs.Client
	queueUrl  string
	tokenAuth bool

	workerCount int
	handler     func(ctx context.Context, msg *types.Message) error

	cancel context.CancelFunc
}

type SQSServerConfig struct {
	Log       *zap.SugaredLogger
	Tracer    trace.Tracer
	QueueUrl  string
	TokenAuth bool
	// A handler for messages
	Handler func(ctx context.Context, msg *types.Message) error
}

func NewSQSServer(ctx context.Context, opts *SQSServerConfig) (*SQSServer, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	client := sqs.NewFromConfig(cfg)

	// default to 10 workers to process SQS messages for now
	// in future we could expose this so that it can be tuned in production.
	workerCount := 10

	return &SQSServer{
		log:         opts.Log,
		tracer:      opts.Tracer,
		client:      client,
		queueUrl:    opts.QueueUrl,
		tokenAuth:   opts.TokenAuth,
		workerCount: workerCount,
		handler:     opts.Handler,
	}, nil
}

// Start begins polling the SQS Server in a separate goroutine
func (s *SQSServer) Start(ctx context.Context) {
	jobs := make(chan *types.Message)
	ctx, cancel := context.WithCancel(ctx)

	for w := 1; w <= s.workerCount; w++ {
		go s.worker(ctx, w, jobs)
	}

	s.cancel = cancel

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				out, err := s.client.ReceiveMessage(ctx,
					&sqs.ReceiveMessageInput{
						QueueUrl:              &s.queueUrl,
						AttributeNames:        []types.QueueAttributeName{types.QueueAttributeNameAll},
						MessageAttributeNames: []string{"All"},
					})
				if err != nil {
					if !errors.Is(err, context.Canceled) {
						s.log.With(zap.Error(err)).Error("error receiving SQS message, retrying in 10s")
						time.Sleep(10 * time.Second)
					}
					continue
				}
				for _, msg := range out.Messages {
					jobs <- &msg
				}
			}
		}
	}()
}

func (s *SQSServer) Shutdown() {
	s.cancel()
}

func (s *SQSServer) worker(ctx context.Context, id int, messages <-chan *types.Message) {
	for m := range messages {
		s.log.With("msg", m).Info("received message")

		// run the message handler
		err := s.handler(ctx, m)
		if err != nil {
			s.log.With(zap.Error(err)).Error("error handling message")
		} else {
			// if no errors handling, delete the message from the queue
			_, err = s.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      &s.queueUrl,
				ReceiptHandle: m.ReceiptHandle,
			})

			if err != nil {
				s.log.With(zap.Error(err)).Error("error deleting message")
			}
		}

	}
}

func (s *SQSServer) QueueUrl() string {
	return s.queueUrl
}

func (c *Collector) HandleSQSMessage(ctx context.Context, msg *types.Message) error {
	var e recommendations.AWSEvent

	err := json.Unmarshal([]byte(*msg.Body), &e)
	if err != nil {
		return errors.Wrap(err, "unmarshling SQS message body")
	}
	var token *tokens.Token

	if c.TransportSQSTokenAuth {

		tokenAttr := msg.MessageAttributes["x-iamzero-token"]
		tokenID := tokenAttr.StringValue
		c.log.With("tokenID", tokenID).Info("looking up token")
		if tokenID == nil {
			return errors.New("IAM Zero token was not found in SQS message attributes (it must be passed as the x-iamzero-token attribute)")
		}

		token, err = c.tokenStore.Get(ctx, *tokenID)
		if err != nil {
			return errors.Wrap(err, "retrieving token from tokenStore")
		}
		if token == nil {
			return errors.New("token not found")
		}
	}

	detective := events.NewDetective(events.DetectiveOpts{
		Log:     c.log,
		Storage: c.storage,
		Auditor: c.auditor,
	})

	_, err = detective.AnalyseEvent(e)

	if err != nil {
		return err
	}
	return nil
}
