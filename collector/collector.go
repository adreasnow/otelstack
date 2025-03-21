package collector

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

type Collector struct {
	Ports   map[int]nat.Port
	config  string
	Network *testcontainers.DockerNetwork
	Name    string
}

func (c *Collector) Start(ctx context.Context, jaegerName string, seqName string) (error, func(context.Context) error) {
	emptyFunc := func(context.Context) error { return nil }
	var err error

	c.Ports = make(map[int]nat.Port)

	if c.Network == nil {
		c.Network, err = network.New(ctx)
		if err != nil {
			return errors.Join(fmt.Errorf("could not create network"), err), emptyFunc
		}
		defer c.Network.Remove(ctx)
	}

	c.generateConfig(jaegerName, seqName)

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "docker-hub.artifactory.xero-support.com/otel/opentelemetry-collector:0.117.0",
			ExposedPorts: []string{"4317/tcp", "13133/tcp"},
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
		return errors.Join(fmt.Errorf("otel collector could not start"), err), emptyFunc
	}

	c.Name, err = container.Name(ctx)
	if err != nil {
		return errors.Join(fmt.Errorf("could not get container name"), err), emptyFunc
	}
	c.Name = c.Name[1:]

	for _, portNum := range []int{4317, 13133} {
		c.Ports[portNum], err = container.MappedPort(ctx, nat.Port(fmt.Sprintf("%d", portNum)))
		if err != nil {
			return errors.Join(fmt.Errorf("the port %d could not be retrieved", portNum), err), emptyFunc
		}
	}

	return nil, func(ctx context.Context) error {
		return container.Terminate(ctx, testcontainers.StopTimeout(time.Second*30))
	}
}

func (c *Collector) generateConfig(jaegerName string, seqName string) {
	c.config = fmt.Sprintf(`
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317

processors:
  batch:

exporters:
  otlp:
    endpoint: %s:4317
    tls:
      insecure: true

  otlphttp:
    endpoint: http://%s/ingest/otlp

  debug:
    verbosity: basic

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
      exporters: [otlp, debug]

    logs:
      receivers: [otlp]
      exporters: [otlphttp, debug]

    metrics:
      receivers: [otlp]
      exporters: [debug]
`, jaegerName, seqName)
}

// func (c *Collector) generateConfig(jaegerPort nat.Port, seqPort nat.Port) {
// 	c.config = fmt.Sprintf(`
// 	receivers:
//   otlp:
//     protocols:
//       grpc:
//         endpoint: 0.0.0.0:4317

// processors:
//   batch:

// exporters:
//   otlp:
//     endpoint: jaeger:%d
//     tls:
//       insecure: true

//   otlphttp:
//     endpoint: http://seq:%d/ingest/otlp

//   debug:
//     verbosity: basic

// extensions:
//   health_check:
//     endpoint: "0.0.0.0:13133"
//     path: "/health/status"
//     check_collector_pipeline:
//       enabled: true
//       interval: "10s"
//       exporter_failure_threshold: 5

// service:
//   extensions: [health_check]
//   pipelines:
//     traces:
//       receivers: [otlp]
//       exporters: [otlp, debug]

//     logs:
//       receivers: [otlp]
//       exporters: [otlphttp, debug]

//     metrics:
//       receivers: [otlp]
//       exporters: [debug]
// `, jaegerPort.Int(), seqPort.Int())
// }
