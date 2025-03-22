package jaeger

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var serviceName = "test.service"

func TestJaegerStart(t *testing.T) {
	t.Parallel()
	j := Jaeger{}
	shutdownFunc, err := j.Start(t.Context())
	require.NoError(t, err, "jaeger must be able to start")
	t.Cleanup(func() {
		if err := shutdownFunc(t.Context()); err != nil {
			t.Logf("error shutting down jaeger: %v", err)
		}
	})

	endpoint := fmt.Sprintf("http://localhost:%d", j.Ports[16686].Int())
	t.Logf("using endpoint: %s", endpoint)

	resp, err := http.Get(endpoint)
	require.NoError(t, err, "must be able to call jaeger")
	assert.Equal(t, 200, resp.StatusCode, "request should be 200")
}

func TestGetTraces(t *testing.T) {
	j := Jaeger{}
	shutdownFunc, err := j.Start(t.Context())
	require.NoError(t, err, "jaeger must be able to start")
	t.Cleanup(func() {
		if err := shutdownFunc(t.Context()); err != nil {
			t.Logf("error shutting down jaeger: %v", err)
		}
	})

	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", fmt.Sprintf("http://localhost:%d", j.Ports[4317].Int()))

	{ // set up otel tracer
		exporter, err := otlptracegrpc.New(t.Context())
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
	}

	{ // send trace
		tracer := otel.Tracer(serviceName)
		_, span := tracer.Start(t.Context(), "test.segment")
		time.Sleep(time.Second * 1)
		span.End()
		time.Sleep(time.Millisecond * 500)
	}

	traces, err := j.GetTraces(t.Context(), 5, serviceName)
	require.NoError(t, err, "must be able to get traces")
	require.Len(t, traces.Data, 1)
	require.Len(t, traces.Data[0].Spans, 1)
	assert.Equal(t, "test.segment", traces.Data[0].Spans[0].OperationName)
}
