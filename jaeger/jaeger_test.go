package jaeger

import (
	"context"
	"fmt"
	"net/http"
	"os"
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

func TestJaegerStart(t *testing.T) {
	t.Parallel()
	j := Jaeger{}
	shutdownFunc, err := j.Start(t.Context())
	require.NoError(t, err, "jaeger must be able to start")
	t.Cleanup(func() {
		if err := shutdownFunc(t.Context()); err != nil {
			// do nothing
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
			// do nothing
		}
	})

	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", fmt.Sprintf("http://localhost:%d", j.Ports[4317].Int()))
	t.Setenv("OTEL_SERVICE_NAME", "test.mapper")

	{ // set up otel tracer
		exporter, err := otlptracegrpc.New(t.Context())
		require.NoError(t, err, "must be able to set up exporter")

		resources, err := resource.New(t.Context(), resource.WithAttributes(attribute.String("service.name", os.Getenv("OTEL_SERVICE_NAME"))))
		require.NoError(t, err, "must be able to set up resources")

		otel.SetTracerProvider(
			sdktrace.NewTracerProvider(
				sdktrace.WithSampler(sdktrace.AlwaysSample()),
				sdktrace.WithBatcher(exporter),
				sdktrace.WithResource(resources),
			),
		)

		t.Cleanup(func() {
			if err := exporter.Shutdown(context.Background()); err != nil {
				// do nothing
			}
		})
	}

	{ // send trace
		tracer := otel.Tracer(os.Getenv("OTEL_SERVICE_NAME"))
		_, span := tracer.Start(t.Context(), "test.segment")
		time.Sleep(time.Second * 3)
		span.End()
		time.Sleep(time.Second * 3)
	}

	traces, err := j.GetTraces(t.Context())
	require.NoError(t, err, "must be able to get traces")
	require.Len(t, traces.Data, 1)
	require.Len(t, traces.Data[0].Spans, 1)
	assert.Equal(t, "test.segment", traces.Data[0].Spans[0].OperationName)
}
