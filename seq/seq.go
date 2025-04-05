// Package seq holds the resources needed to start a Seq testcontainer.
package seq

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Seq hold the testcontainer, ports and network used by Seq. If instantiating yourself,
// be sure to populate Seq.Network, otherwise a new network will be generated.
type Seq struct {
	Ports   map[int]nat.Port
	Network *testcontainers.DockerNetwork
	Name    string
}

// Start starts the Seq container.
func (s *Seq) Start(ctx context.Context) (func(context.Context) error, error) {
	emptyFunc := func(context.Context) error { return nil }
	var err error

	s.Ports = make(map[int]nat.Port)

	if s.Network == nil {
		s.Network, err = network.New(ctx)
		if err != nil {
			return emptyFunc, fmt.Errorf("seq: network not provided and could not create a new one: %w", err)
		}
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "datalust/seq:2024.3",
			ExposedPorts: []string{"80/tcp", "5341/tcp"},
			Networks:     []string{s.Network.Name},
			WaitingFor:   wait.ForLog("Seq listening on"),
			Env:          map[string]string{"ACCEPT_EULA": "Y"},
		},
		Started: true,
	})
	if err != nil {
		return emptyFunc, fmt.Errorf("seq: could not start the testcontainer: %w", err)
	}

	s.Name, err = container.Name(ctx)
	if err != nil {
		return emptyFunc, fmt.Errorf("seq: could not read the name of the container from the testcontainer: %w", err)
	}
	s.Name = s.Name[1:]

	for _, portNum := range []int{80, 5341} {
		s.Ports[portNum], err = container.MappedPort(ctx, nat.Port(fmt.Sprintf("%d", portNum)))
		if err != nil {
			return emptyFunc, fmt.Errorf("seq: could not retrieve port %d from the testcontainer: %w", portNum, err)
		}
	}

	return func(ctx context.Context) error {
		return container.Terminate(ctx, testcontainers.StopTimeout(time.Second*30))
	}, nil
}
