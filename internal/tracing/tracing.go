package tracing

import (
	"context"
	"flag"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

type TracingFactory struct {
	Enabled bool
}

func NewFactory() *TracingFactory {
	return &TracingFactory{}
}

func (f *TracingFactory) AddFlags(fs *flag.FlagSet) {
	fs.BoolVar(&f.Enabled, "tracing-enabled", false, "enable OpenTelemetry tracing")
}

func (f *TracingFactory) InitializeTracer(ctx context.Context) (trace.Tracer, error) {
	var tracer trace.Tracer
	if f.Enabled {
		if err := setupOtel(ctx); err != nil {
			return nil, err
		}
		tracer = otel.Tracer("iamzero.dev/server")
	} else {
		tracer = trace.NewNoopTracerProvider().Tracer("")
	}
	return tracer, nil
}

// setupOtel sets up opentelemetry tracing
func setupOtel(ctx context.Context) error {
	endpoint := "localhost:4317"
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("iamzero-server"),
		),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create resource")
	}

	// Set up a trace exporter
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithDialOption(grpc.FailOnNonTempDialError(true)),
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithDialOption(grpc.WithReturnConnectionError()),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create trace exporter")
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

	return nil
}
