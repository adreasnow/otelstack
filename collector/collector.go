// Package collector holds the resources needed to start an OTEL collector testcontainer
package collector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Collector hold the testcontainer, ports and network used by the OTEL collector.
// If instantiating yourself, be sure to populate Collector.Network, otherwise a new network will be generated.
type Collector struct {
	Ports   map[int]nat.Port
	config  string
	Network *testcontainers.DockerNetwork
	Name    string
}

// Start starts the OTEL collector container.
func (c *Collector) Start(ctx context.Context, jaegerName string, seqName string) (func(context.Context) error, error) {
	emptyFunc := func(context.Context) error { return nil }
	var err error

	c.Ports = make(map[int]nat.Port)

	if c.Network == nil {
		c.Network, err = network.New(ctx)
		if err != nil {
			return emptyFunc, errors.WithMessage(err, "collector: network not provided and could not create a new one")
		}
	}

	c.generateConfig(jaegerName, seqName)

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "otel/opentelemetry-collector:0.117.0",
			ExposedPorts: []string{"4317/tcp", "4318/tcp", "13133/tcp"},
			Networks:     []string{c.Network.Name},
			WaitingFor:   wait.ForLog("Everything is ready. Begin running and processing data"),
			Files: []testcontainers.ContainerFile{{
				ContainerFilePath: "/etc/otelcol/config.yaml",
				Reader:            strings.NewReader(c.config),
				FileMode:          0644,
			}},
		},
		Started: true,
	})
	if err != nil {
		return emptyFunc, errors.WithMessage(err, "collector: could not start the testcontainer")
	}

	c.Name, err = container.Name(ctx)
	if err != nil {
		return emptyFunc, errors.WithMessage(err, "collector: could not read the name of the container from the testcontainer")
	}
	c.Name = c.Name[1:]

	for _, portNum := range []int{4317, 4318, 13133} {
		c.Ports[portNum], err = container.MappedPort(ctx, nat.Port(fmt.Sprintf("%d", portNum)))
		if err != nil {
			return emptyFunc, errors.Wrapf(err, "collector: could not retrieve port %d from the testcontainer", portNum)
		}
	}

	return func(ctx context.Context) error {
		return container.Terminate(ctx, testcontainers.StopTimeout(time.Second*30))
	}, nil
}

func (c *Collector) generateConfig(jaegerName string, seqName string) {
	c.config = fmt.Sprintf(`
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

exporters:
  otlp:
    endpoint: %s:4317
    tls:
      insecure: true

  otlphttp/logs:
    endpoint: http://%s/ingest/otlp

  prometheus:
    endpoint: "0.0.0.0:8889"
    send_timestamps: true
    metric_expiration: 180m
    resource_to_telemetry_conversion:
      enabled: true

extensions:
  health_check:
    endpoint: "0.0.0.0:13133"
    path: "/health/status"
    check_collector_pipeline:
      enabled: true
      interval: "10s"
      exporter_failure_threshold: 5

service:
  extensions: [health_check]
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [otlp]

    logs:
      receivers: [otlp]
      exporters: [otlphttp/logs]

    metrics:
      receivers: [otlp]
      exporters: [prometheus]
`, jaegerName, seqName)
}
