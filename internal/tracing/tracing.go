package tracing

import (
	"context"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func NewTracingService(ctx context.Context, log *zap.SugaredLogger) (trace.TracerProvider, error) {
	endpoint := "localhost:4317"
	log.With("oltpEndpoint", endpoint).Info("configuring tracing")
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("iamzero-server"),
		),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create resource")
	}

	// Set up a trace exporter
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithDialOption(grpc.FailOnNonTempDialError(true)),
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithDialOption(grpc.WithReturnConnectionError()),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create trace exporter")
	}

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tracerProvider, nil
}

// // Shutdown must be called by the server before shutting down to send any remaining traces and close the connection
// func (t *TracingService) Shutdown(ctx context.Context) error {
// 	return t.provider.Shutdown(ctx)
// }
