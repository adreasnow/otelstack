// Package jaeger holds the resources needed to start a Jaeger testcontainer container.
package jaeger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Jaeger hold the testcontainer, ports and network used by Jaeger. If instansiating yourself,
// be sure to popule Jaeger.Network, otherwise a new network will be generated.
type Jaeger struct {
	Ports   map[int]nat.Port
	Network *testcontainers.DockerNetwork
	Name    string
}

type jaegerTraces struct {
	Data []struct {
		TraceID string `json:"traceID"`
		Spans   []struct {
			TraceID       string `json:"traceID"`
			SpanID        string `json:"spanID"`
			OperationName string `json:"operationName"`
		} `json:"spans"`
	} `json:"data"`
}

// GetTraces takes in a service names and returns the last n traces corresponding to that service.
func (j *Jaeger) GetTraces(ctx context.Context, maxTraces int, service string) (traces jaegerTraces, err error) {
	endpoint := fmt.Sprintf("http://localhost:%d/api/traces?service=%s&limit=%d", j.Ports[16686].Int(), url.QueryEscape(service), maxTraces)

	resp, err := http.Get(endpoint)
	if err != nil {
		err = fmt.Errorf("must be able to get traces from jaeger: %v", err)
		return
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf("must be a 200 response, got %d", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("must be able to get set body from seq response: %v", err)
		return
	}

	err = json.Unmarshal(body, &traces)
	if err != nil {
		err = fmt.Errorf("must be able unmarshal traces: %v", err)
		return
	}

	return
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
			Image:        "jaegertracing/all-in-one:1.65.0",
			ExposedPorts: []string{"16686/tcp", "14268/tcp", "6831/tcp", "4317/tcp"},
			Networks:     []string{j.Network.Name},
			WaitingFor:   wait.ForLog(`"msg":"Health Check state change","status":"ready"`),
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

	for _, portNum := range []int{16686, 14268, 6831, 4317} {
		j.Ports[portNum], err = container.MappedPort(ctx, nat.Port(fmt.Sprintf("%d", portNum)))
		if err != nil {
			return emptyFunc, fmt.Errorf("the port %d could not be retrieved: %v", portNum, err)
		}
	}

	return func(ctx context.Context) error {
		return container.Terminate(ctx, testcontainers.StopTimeout(time.Second*30))
	}, nil
}
