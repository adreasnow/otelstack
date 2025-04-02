// Package jaeger holds the resources needed to start a Jaeger testcontainer container.
package jaeger

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Jaeger hold the testcontainer, ports and network used by Jaeger. If instantiating yourself,
// be sure to populate Jaeger.Network, otherwise a new network will be generated.
type Jaeger struct {
	Ports   map[int]nat.Port
	Network *testcontainers.DockerNetwork
	Name    string
}

// Start starts the Jaeger container.
func (j *Jaeger) Start(ctx context.Context) (func(context.Context) error, error) {
	emptyFunc := func(context.Context) error { return nil }
	var err error

	j.Ports = make(map[int]nat.Port)

	if j.Network == nil {
		j.Network, err = network.New(ctx)
		if err != nil {
			return emptyFunc, fmt.Errorf("could not create network: %v", err)
		}
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "jaegertracing/jaeger:latest",
			ExposedPorts: []string{"16686/tcp", "4318/tcp"},
			Networks:     []string{j.Network.Name},
			WaitingFor:   wait.ForListeningPort("16686/tcp"),
			Cmd:          []string{"--config", "/etc/jaeger/config.yaml"},
			Files: []testcontainers.ContainerFile{{
				ContainerFilePath: "/etc/jaeger/config.yaml",
				Reader:            strings.NewReader(config),
				FileMode:          0644,
			}},
		},
		Started: true,
	})
	if err != nil {
		return emptyFunc, fmt.Errorf("jaeger could not start: %v", err)
	}

	j.Name, err = container.Name(ctx)
	if err != nil {
		return emptyFunc, fmt.Errorf("could not get container name: %v", err)
	}
	j.Name = j.Name[1:]

	for _, portNum := range []int{16686, 4318} {
		j.Ports[portNum], err = container.MappedPort(ctx, nat.Port(fmt.Sprintf("%d", portNum)))
		if err != nil {
			return emptyFunc, fmt.Errorf("the port %d could not be retrieved: %v", portNum, err)
		}
	}

	return func(ctx context.Context) error {
		return container.Terminate(ctx, testcontainers.StopTimeout(time.Second*30))
	}, nil
}

var config = `service:
  extensions: [jaeger_storage, jaeger_query]
  pipelines:
    traces:
      receivers: [otlp]
      processors: []
      exporters: [jaeger_storage_exporter]

extensions:
  jaeger_query:
    storage:
      traces: some_storage

  jaeger_storage:
    backends:
      some_storage:
        memory:
          max_traces: 100000

receivers:
  otlp:
    protocols:
      grpc:
        endpoint: "0.0.0.0:4317"
      http:
        endpoint: "0.0.0.0:4318"

exporters:
  jaeger_storage_exporter:
    trace_storage: some_storage
`
