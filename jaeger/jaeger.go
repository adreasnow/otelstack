package jaeger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

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

func (j *Jaeger) GetTraces(ctx context.Context) (traces jaegerTraces, err error) {
	service := os.Getenv("OTEL_SERVICE_NAME")
	endpoint := fmt.Sprintf("http://localhost:%d/api/traces?service=%s&limit=5", j.Ports[16686].Int(), service)
	resp, err := http.Get(endpoint)
	if err != nil {
		err = errors.Join(fmt.Errorf("must be able to get traces from jaeger"), err)
		return
	}

	if resp.StatusCode != 200 {
		err = errors.Join(fmt.Errorf("must be a 200 response"), err)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		err = errors.Join(fmt.Errorf("must be able to get set body from seq response"), err)
		return
	}

	err = json.Unmarshal(body, &traces)
	if err != nil {
		err = errors.Join(fmt.Errorf("must be able unmarshal traces"), err)
		return
	}

	return
}

func (j *Jaeger) Start(ctx context.Context) (error, func(context.Context) error) {
	emptyFunc := func(context.Context) error { return nil }
	var err error

	j.Ports = make(map[int]nat.Port)

	if j.Network == nil {
		j.Network, err = network.New(ctx)
		if err != nil {
			return errors.Join(fmt.Errorf("could not create network"), err), emptyFunc
		}
		defer j.Network.Remove(ctx)
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
		return errors.Join(fmt.Errorf("jaeger could not start"), err), emptyFunc
	}

	j.Name, err = container.Name(ctx)
	if err != nil {
		return errors.Join(fmt.Errorf("could not get container name"), err), emptyFunc
	}
	j.Name = j.Name[1:]

	for _, portNum := range []int{16686, 14268, 6831, 4317} {
		j.Ports[portNum], err = container.MappedPort(ctx, nat.Port(fmt.Sprintf("%d", portNum)))
		if err != nil {
			return errors.Join(fmt.Errorf("the port %d could not be retrieved", portNum), err), emptyFunc
		}
	}

	return nil, func(ctx context.Context) error {
		return container.Terminate(ctx, testcontainers.StopTimeout(time.Second*30))
	}
}
