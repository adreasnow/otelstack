package prometheus

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/adreasnow/otelstack/collector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/network"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

var serviceName = "test service"

func TestGetMetrics(t *testing.T) {
	network, err := network.New(t.Context())
	require.NoError(t, err, "must be able to create network")
	t.Cleanup(func() {
		if err := network.Remove(t.Context()); err != nil {
			t.Logf("could not shut down network: %v", err)
		}
	})

	c := collector.Collector{
		Network: network,
	}
	collectorShutdownFunc, err := c.Start(t.Context(), "jaeger", "seq")
	require.NoError(t, err, "collector must be able to start")
	t.Cleanup(func() {
		if err := collectorShutdownFunc(t.Context()); err != nil {
			t.Logf("error shutting down collector: %v", err)
		}
	})

	p := Prometheus{
		Network: network,
	}
	prometheusShutdownFunc, err := p.Start(t.Context(), c.Name)
	require.NoError(t, err, "prometheus must be able to start")
	t.Cleanup(func() {
		if err := prometheusShutdownFunc(t.Context()); err != nil {
			t.Logf("error shutting down prometheus: %v", err)
		}
	})

	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", fmt.Sprintf("http://localhost:%d", c.Ports[4317].Int()))

	exporter, err := otlpmetricgrpc.New(t.Context())
	require.NoError(t, err, "must be able to set up exporter")

	resources, err := resource.New(t.Context(), resource.WithAttributes(attribute.String("service.name", serviceName)))
	require.NoError(t, err, "must be able to set up resources")

	otel.SetMeterProvider(
		sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(resources),
			sdkmetric.WithReader(
				sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(time.Second)),
			),
		),
	)

	t.Cleanup(func() {
		if err := exporter.Shutdown(context.Background()); err != nil {
			t.Logf("error shutting down exporter: %v", err)
		}
	})

	meter := otel.Meter(serviceName)
	_, err = meter.Int64ObservableGauge("goroutine.count",
		metric.WithUnit("goroutine"),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			o.Observe(int64(runtime.NumGoroutine()))
			return nil
		}),
	)
	require.NoError(t, err, "must be able to set up the goroutines meter")

	time.Sleep(time.Second * 10)

	m, endpoint, err := p.GetMetrics(3, 30, "goroutine_count", serviceName, time.Second*30)
	require.NoError(t, err, "must be able to get metrics")

	assert.NotEmpty(t, endpoint, "must return an endpoint")
	assert.GreaterOrEqual(t, len(m.Values), 3)
	num, err := strconv.Atoi(m.Values[0][1].(string))
	require.NoError(t, err, "must be able to parse the metric value")

	assert.Greater(t, num, 2)
}
