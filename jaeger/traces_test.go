package jaeger

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var serviceName = "test service"

func TestGetTraces(t *testing.T) {
	j := Jaeger{}
	shutdownFunc, err := j.Start(t.Context())
	require.NoError(t, err, "jaeger must be able to start")
	t.Cleanup(func() {
		if err := shutdownFunc(t.Context()); err != nil {
			t.Logf("error shutting down jaeger: %v", err)
		}
	})

	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", fmt.Sprintf("http://localhost:%d", j.Ports[4318].Int()))

	// set up otel tracer
	exporter, err := otlptracehttp.New(t.Context())
	require.NoError(t, err, "must be able to set up exporter")

	resources, err := resource.New(t.Context(), resource.WithAttributes(attribute.String("service.name", serviceName)))
	require.NoError(t, err, "must be able to set up resources")

	otel.SetTracerProvider(
		sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithSyncer(exporter),
			sdktrace.WithResource(resources),
		),
	)

	t.Cleanup(func() {
		if err := exporter.Shutdown(context.Background()); err != nil {
			t.Logf("error shutting down exporter: %v", err)
		}
	})

	{ // send trace
		tracer := otel.Tracer(serviceName)
		_, span := tracer.Start(t.Context(), "test.segment")
		time.Sleep(time.Millisecond * 100)
		span.End()
	}

	time.Sleep(time.Second * 1)

	traces, err := j.GetTraces(1, 30, serviceName)
	require.NoError(t, err, "must be able to get traces")

	require.Len(t, traces, 1)
	require.Len(t, traces[0].Spans, 1)
	assert.Equal(t, "test.segment", traces[0].Spans[0].OperationName)
}
