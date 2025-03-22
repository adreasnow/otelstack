package otelstack

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"

	otelLogGlobal "go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func setupOTELHTTP(t *testing.T) {
	resources, err := resource.New(t.Context(), resource.WithAttributes(attribute.String("service.name", serviceName)))
	require.NoError(t, err, "resources must be created")

	logExporter, err := otlploghttp.New(t.Context())
	require.NoError(t, err, "log exporter must be started")
	traceExporter, err := otlptracehttp.New(t.Context())
	require.NoError(t, err, "trace exporter must be started")

	t.Cleanup(func() {
		if err := logExporter.Shutdown(context.Background()); err != nil {
			t.Logf("error shutting down stack: %v", err)
		}
		if err := traceExporter.Shutdown(context.Background()); err != nil {
			t.Logf("error shutting down stack: %v", err)
		}
	})

	otelLogGlobal.SetLoggerProvider(
		sdklog.NewLoggerProvider(
			sdklog.WithProcessor(
				sdklog.NewSimpleProcessor(logExporter),
			),
			sdklog.WithResource(resources),
		),
	)

	otel.SetTracerProvider(
		sdktrace.NewTracerProvider(
			sdktrace.WithSyncer(traceExporter),
			sdktrace.WithResource(resources),
		),
	)
}

func setupOTELgRPC(t *testing.T) {
	resources, err := resource.New(t.Context(), resource.WithAttributes(attribute.String("service.name", serviceName)))
	require.NoError(t, err, "resources must be created")

	logExporter, err := otlploggrpc.New(t.Context())
	require.NoError(t, err, "log exporter must be started")
	traceExporter, err := otlptracegrpc.New(t.Context())
	require.NoError(t, err, "trace exporter must be started")

	t.Cleanup(func() {
		if err := logExporter.Shutdown(context.Background()); err != nil {
			t.Logf("error shutting down stack: %v", err)
		}
		if err := traceExporter.Shutdown(context.Background()); err != nil {
			t.Logf("error shutting down stack: %v", err)
		}
	})

	otelLogGlobal.SetLoggerProvider(
		sdklog.NewLoggerProvider(
			sdklog.WithProcessor(
				sdklog.NewSimpleProcessor(logExporter),
			),
			sdklog.WithResource(resources),
		),
	)

	otel.SetTracerProvider(
		sdktrace.NewTracerProvider(
			sdktrace.WithSyncer(traceExporter),
			sdktrace.WithResource(resources),
		),
	)
}

func TestStartStack(t *testing.T) {
	testData := []struct {
		name string
	}{{"gRPC"}, {"HTTP"}}
	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			s := stack{}
			shutdownFunc, err := s.Start(t.Context())
			require.NoError(t, err, "stack must be able to start")
			t.Cleanup(func() {
				if err := shutdownFunc(context.Background()); err != nil {
					t.Logf("error shutting down stack: %v", err)
				}
			})

			switch tt.name {
			case "gRPC":
				s.SetTestEnvGRPC(t)
				setupOTELgRPC(t)
			case "HTTP":
				s.SetTestEnvHTTP(t)
				setupOTELHTTP(t)
			}

			{ // send data
				tracer := otel.Tracer(serviceName)
				ctx, span := tracer.Start(t.Context(), "test.segment")
				trace.SpanFromContext(ctx).SetAttributes(attribute.String("test.key", "test_value"))

				record := log.Record{}
				record.SetTimestamp(time.Now())
				record.SetBody(log.StringValue("test message"))
				record.SetSeverity(log.SeverityError)
				record.SetSeverityText("ERROR")
				record.AddAttributes(log.String("SpanID", span.SpanContext().SpanID().String()))
				record.AddAttributes(log.String("TraceID", span.SpanContext().TraceID().String()))
				otelLogGlobal.GetLoggerProvider().
					Logger(serviceName).
					Emit(t.Context(), record)
				span.End()
			}

			time.Sleep(time.Millisecond * 100)

			t.Run("test traces", func(t *testing.T) {
				traces, err := s.Jaeger.GetTraces(t.Context(), 5, serviceName)
				require.NoError(t, err, "must be able to get traces")
				require.Len(t, traces.Data, 1)
				require.Len(t, traces.Data[0].Spans, 1)
				assert.Equal(t, "test.segment", traces.Data[0].Spans[0].OperationName)
			})

			t.Run("test logs", func(t *testing.T) {
				events, err := s.Seq.GetEvents(t.Context(), 5)
				require.NoError(t, err)
				require.Len(t, events, 1)
				require.Len(t, events[0].MessageTemplateTokens, 1)
				assert.Equal(t, "test message", events[0].MessageTemplateTokens[0].Text)
			})
		})
	}
}
