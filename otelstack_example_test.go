package otelstack

import (
	"context"
	"testing"
	"time"

	"github.com/adreasnow/otelstack/collector"
	"github.com/adreasnow/otelstack/jaeger"
	"github.com/adreasnow/otelstack/seq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	otelLogGlobal "go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/trace"
)

var serviceName = "test.service"

func TestExampleSetupStack(t *testing.T) {
	stack := New()
	shutdownFunc, err := stack.Start(t.Context())
	require.NoError(t, err, "the stack must start up")

	// Ve sure to defer shutdown of the stack
	t.Cleanup(func() {
		if err := shutdownFunc(context.Background()); err != nil {
			t.Logf("error shutting down stack: %v", err)
		}
	})

	// For optionally setting OTEL_EXPORTER_OTLP_ENDPOINT
	stack.SetTestEnvGRPC(t)
	// can also use stack.SetTestEnvHTTP(t) for the http endpoint

	// Ports can be accessed as such
	t.Logf("Seq ui: http://localhost:%d", stack.Seq.Ports[80].Int())
	t.Logf("Jaeger ui: http://localhost:%d", stack.Seq.Ports[16686].Int())

	t.Logf("OTEL gRPC endpoint: http://localhost:%d", stack.Collector.Ports[4317].Int())

	// Continue to initialise your own otel setup here
	shutdown := setupOTELgRPC(t)

	// As a backup in case something happens before shutdown is manually called
	t.Cleanup(func() {
		// t.Context() will be closed before this is called, so be sure to use a new context
		if err := shutdown(context.Background()); err != nil {
			t.Logf("error shutting down otel: %v", err)
		}
	})

	{ // send some traces and logs to otel
		tracer := otel.Tracer(serviceName)
		ctx, span := tracer.Start(t.Context(), "test-segment")
		trace.SpanFromContext(ctx).SetAttributes(attribute.String("test-key", "test-value"))
		// Do some work
		time.Sleep(time.Second * 1)
		record := log.Record{}
		record.SetTimestamp(time.Now())
		record.SetBody(log.StringValue("test message"))
		record.SetSeverity(log.SeverityError)
		record.SetSeverityText("ERROR")
		otelLogGlobal.GetLoggerProvider().
			Logger(serviceName).
			Emit(t.Context(), record)
		span.End()
	}

	// Shut down OTEL to allow everything to propagate
	err = shutdown(context.Background())
	require.NoError(t, err)
	time.Sleep(time.Second * 1)

	// Get traces from Jaeger (this can take a while to propagate from
	// span --> collector --> jaeger, so we'll keep trying for a while)
	traces, err := stack.Jaeger.GetTraces(1, 10, serviceName)
	require.NoError(t, err, "must be able to get traces")

	require.NoError(t, err, "must be able to get traces")
	assert.Equal(t, "test-segment", traces[0].Spans[0].OperationName)

	// Get log events from Seq
	events, err := stack.Seq.GetEvents(1, 10)
	require.NoError(t, err)
	assert.Equal(t, "test message", events[0].MessageTemplateTokens[0].Text)

}

// Containers can also be started independently if needed, though they won't
// have OTEL ingestion
// A if a tescontainer network isn't provided in the struct, one will be created
func TestExampleSetupContainers(t *testing.T) {
	t.Run("test setup seq", func(t *testing.T) {
		t.Parallel()
		seq := seq.Seq{}
		shutdownFunc, err := seq.Start(t.Context())
		require.NoError(t, err, "the container must start up")

		// Be sure to defer shutdown of the stack
		t.Cleanup(func() {
			if err := shutdownFunc(context.Background()); err != nil {
				t.Logf("error shutting down seq: %v", err)
			}
		})

		t.Logf("Seq ui: http://localhost:%d", seq.Ports[80].Int())
	})

	t.Run("test setup jaeger", func(t *testing.T) {
		t.Parallel()
		jaeger := jaeger.Jaeger{}
		shutdownFunc, err := jaeger.Start(t.Context())
		require.NoError(t, err, "the container must start up")

		// Be sure to defer shutdown of the stack
		t.Cleanup(func() {
			if err := shutdownFunc(context.Background()); err != nil {
				t.Logf("error shutting down jaeger: %v", err)
			}
		})

		t.Logf("Jaeger ui: http://localhost:%d", jaeger.Ports[16686].Int())
	})

	t.Run("test setup collector", func(t *testing.T) {
		t.Parallel()
		collector := collector.Collector{}

		// hostnames of the seq and jaeger services must be provided for generating
		// the OTEL collector config. This requires all pods to be on the same
		// testcontainer network (set in their structs.)
		shutdownFunc, err := collector.Start(t.Context(), "jaeger", "seq")
		require.NoError(t, err, "the container must start up")

		// Be sure to defer shutdown of the stack
		t.Cleanup(func() {
			if err := shutdownFunc(context.Background()); err != nil {
				t.Logf("error shutting down collector: %v", err)
			}
		})

		t.Logf("Collector gRPC endpoint: http://localhost:%d", collector.Ports[4317].Int())
	})
}
