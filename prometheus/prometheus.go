// Package prometheus holds the resources needed to start a Prometheus testcontainer container.
package prometheus

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

// Prometheus holds the testcontainer, ports and network used by Jaeger. If instantiating yourself,
// be sure to populate Jaeger.Network, otherwise a new network will be generated.
type Prometheus struct {
	Ports   map[int]nat.Port
	Network *testcontainers.DockerNetwork
	Name    string
	config  string
}

// Start starts the Prometheus container.
func (p *Prometheus) Start(ctx context.Context, collectorName string) (func(context.Context) error, error) {
	emptyFunc := func(context.Context) error { return nil }
	var err error

	p.Ports = make(map[int]nat.Port)

	if p.Network == nil {
		p.Network, err = network.New(ctx)
		if err != nil {
			return emptyFunc, fmt.Errorf("could not create network: %v", err)
		}
	}

	p.generateConfig(collectorName)

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "prom/prometheus:v3.2.1",
			ExposedPorts: []string{"9090/tcp"},
			Networks:     []string{p.Network.Name},
			WaitingFor:   wait.ForListeningPort("9090/tcp"),
			Files: []testcontainers.ContainerFile{{
				ContainerFilePath: "/etc/prometheus/prometheus.yml",
				Reader:            strings.NewReader(p.config),
				FileMode:          0644,
			}},
		},
		Started: true,
	})
	if err != nil {
		return emptyFunc, fmt.Errorf("prometheus could not start: %v", err)
	}

	p.Name, err = container.Name(ctx)
	if err != nil {
		return emptyFunc, fmt.Errorf("could not get container name: %v", err)
	}
	p.Name = p.Name[1:]

	for _, portNum := range []int{9090} {
		p.Ports[portNum], err = container.MappedPort(ctx, nat.Port(fmt.Sprintf("%d", portNum)))
		if err != nil {
			return emptyFunc, fmt.Errorf("the port %d could not be retrieved: %v", portNum, err)
		}
	}

	return func(ctx context.Context) error {
		return container.Terminate(ctx, testcontainers.StopTimeout(time.Second*30))
	}, nil
}

func (p *Prometheus) generateConfig(collectorName string) {
	p.config = fmt.Sprintf(`
global:
  scrape_interval: 2s
  evaluation_interval: 2s

scrape_configs:
  - job_name: otel
    static_configs:
      - targets: ["%s:8889"]

otlp:
  keep_identifying_resource_attributes: true
  # Recommended attributes to be promoted to labels.
  promote_resource_attributes:
    - service.instance.id
    - service.name
    - service.namespace
    - service.version

storage:
  tsdb:
    out_of_order_time_window: 10m
`, collectorName)
}
