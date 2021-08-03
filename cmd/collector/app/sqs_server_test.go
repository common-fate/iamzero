package app

import (
	"context"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var mockURL = "https://sqs.ap-southeast-2.amazonaws.com/123456789012/iamzero-test"

func BuildSQSServer(t *testing.T) *SQSServer {
	s, err := NewSQSServer(context.Background(), &SQSServerConfig{
		Log:      zap.NewNop().Sugar(),
		Tracer:   trace.NewNoopTracerProvider().Tracer(""),
		QueueUrl: mockURL,
	})

	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestNewSQSServer_ConfiguresQueueUrl(t *testing.T) {
	s := BuildSQSServer(t)
	assert.Equal(t, s.QueueUrl(), mockURL)
}

func TestWorkerExecutesHandler(t *testing.T) {
	s := BuildSQSServer(t)

	jobs := make(chan *types.Message)
	body := "test message"

	var wg sync.WaitGroup
	wg.Add(1)
	executed := false

	handler := func(ctx context.Context, msg *types.Message) error {
		assert.Equal(t, body, *msg.Body)
		executed = true
		wg.Done()
		return nil
	}

	s.handler = handler

	go s.worker(context.Background(), 0, jobs)

	msg := types.Message{
		Body: &body,
	}
	jobs <- &msg
	wg.Wait()
	assert.True(t, executed)
}
