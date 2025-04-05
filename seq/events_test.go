package seq

import (
	"context"
	"testing"
	"time"

	"github.com/adreasnow/otelstack/collector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/log"
	otelLogGlobal "go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var serviceName = "test-service"

func TestGetEvents(t *testing.T) {
	t.Parallel()

	s := Seq{}
	seqShutdownFunc, err := s.Start(t.Context())
	require.NoError(t, err, "seq must be able to start")
	t.Cleanup(func() {
		if err := seqShutdownFunc(context.Background()); err != nil {
			t.Logf("error shutting down seq: %v", err)
		}
	})

	c := collector.Collector{Network: s.Network}
	collectorShutdownFunc, err := c.Start(t.Context(), "jaeger", s.Name)
	require.NoError(t, err, "seq must be able to start")
	t.Cleanup(func() {
		if err := collectorShutdownFunc(context.Background()); err != nil {
			t.Logf("error shutting down seq: %v", err)
		}
	})

	// set up otel logger
	logExporter, err := otlploggrpc.New(t.Context(),
		otlploggrpc.WithEndpoint("localhost:"+c.Ports[4317].Port()),
		otlploggrpc.WithInsecure(),
	)
	require.NoError(t, err, "must be able to set up exporter")

	resources, err := resource.New(t.Context(), resource.WithAttributes(attribute.String("service.name", serviceName)))
	require.NoError(t, err, "must be able to set up resources")

	otelLogGlobal.SetLoggerProvider(
		sdklog.NewLoggerProvider(
			sdklog.WithProcessor(
				sdklog.NewSimpleProcessor(logExporter),
			),
			sdklog.WithResource(resources),
		),
	)

	t.Cleanup(func() {
		if err := logExporter.Shutdown(context.Background()); err != nil {
			t.Logf("error shutting down log exporter: %v", err)
		}
	})

	// set up otel tracer
	otel.SetTracerProvider(
		sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithResource(resources),
		),
	)

	var span trace.Span
	var ctx context.Context
	{ // send log
		func() {
			tracer := otel.Tracer(serviceName)
			ctx, span = tracer.Start(t.Context(), "segment")
			defer span.End()
			record := log.Record{}
			record.SetTimestamp(time.Now())
			record.SetBody(log.StringValue("this is an error message"))

			record.SetSeverity(log.SeverityError)
			record.SetSeverityText("ERROR")
			record.AddAttributes(
				log.Bool("testBool", true),
				log.Float64("testFloat", 3.14159),
				log.String("error", "otelstack: creating a test error"),
			)
			otelLogGlobal.GetLoggerProvider().Logger("").Emit(ctx, record)
		}()
	}

	events, endpoint, err := s.GetEvents(1, 30)

	require.NoError(t, err, "must be able to get events")
	assert.NotEmpty(t, endpoint, "must return an endpoint")
	require.Len(t, events, 1)

	assert.Equal(t, span.SpanContext().SpanID().String(), events[0].SpanID)
	assert.Equal(t, span.SpanContext().TraceID().String(), events[0].TraceID)

	require.Len(t, events[0].Messages, 1)
	assert.Equal(t, "this is an error message", events[0].Messages[0].Text)

	require.Len(t, events[0].Properties, 3)
	propertiesMap := map[string]Property{}
	for _, p := range events[0].Properties {
		propertiesMap[p.Name] = p
	}
	assert.Equal(t, 3.14159, propertiesMap["testFloat"].Value.(float64))
	assert.True(t, propertiesMap["testBool"].Value.(bool))
	assert.Equal(t, "otelstack: creating a test error", propertiesMap["error"].Value.(string))
}
