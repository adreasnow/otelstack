package otelstack

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"

	otelLogGlobal "go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func setupOTEL(t *testing.T) {
	resources, err := resource.New(t.Context(), resource.WithAttributes(attribute.String("service.name", os.Getenv("OTEL_SERVICE_NAME"))))
	require.NoError(t, err, "resources must be created")

	logExporter, err := otlploggrpc.New(t.Context())
	require.NoError(t, err, "log exporter must be started")
	traceExporter, err := otlptracegrpc.New(t.Context())
	require.NoError(t, err, "trace exporter must be started")

	t.Cleanup(func() {
		if err := logExporter.Shutdown(context.Background()); err != nil {
			// do nothing
		}
		if err := traceExporter.Shutdown(context.Background()); err != nil {
			// do nothing
		}
	})

	otelLogGlobal.SetLoggerProvider(
		sdklog.NewLoggerProvider(
			sdklog.WithProcessor(
				sdklog.NewBatchProcessor(logExporter),
			),
			sdklog.WithResource(resources),
		),
	)

	otel.SetTracerProvider(
		sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(traceExporter),
			sdktrace.WithResource(resources),
		),
	)
}
func TestStartStack(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "test")

	s := stack{}
	shutdownFunc, err := s.Start(t.Context())
	require.NoError(t, err, "start must be able to start")
	t.Cleanup(func() {
		if err := shutdownFunc(t.Context()); err != nil {
			// do nothing
		}
	})

	s.SetTestEnv(t)

	setupOTEL(t)

	t.Run("test-traces", func(t *testing.T) {
		t.Parallel()
		tracer := otel.Tracer(os.Getenv("OTEL_SERVICE_NAME"))
		ctx, span := tracer.Start(t.Context(), "test.segment")
		trace.SpanFromContext(ctx).SetAttributes(attribute.String("test.key", "test_value"))
		time.Sleep(time.Second * 3)
		span.End()
		time.Sleep(time.Second * 3)

		traces, err := s.Jaeger.GetTraces(t.Context())
		require.NoError(t, err, "must be able to get traces")
		require.Len(t, traces.Data, 1)
		require.Len(t, traces.Data[0].Spans, 1)
		assert.Equal(t, "test.segment", traces.Data[0].Spans[0].OperationName)
	})

	t.Run("test-logs", func(t *testing.T) {
		t.Parallel()
		record := log.Record{}
		record.SetTimestamp(time.Now())
		record.SetBody(log.StringValue("test message"))
		record.SetSeverity(log.SeverityError)
		record.SetSeverityText("ERROR")

		otelLogGlobal.GetLoggerProvider().
			Logger(os.Getenv("OTEL_SERVICE_NAME")).
			Emit(t.Context(), record)

		time.Sleep(time.Second * 3)

		events, err := s.Seq.GetEvents(t.Context())
		require.NoError(t, err)
		require.Len(t, events, 1)
		require.Len(t, events[0].MessageTemplateTokens, 1)
		assert.Equal(t, "test message", events[0].MessageTemplateTokens[0].Text)
	})
}
