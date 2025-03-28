// Package seq holds the resources needed to start a Seq testcontainer.
package seq

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// Events holds the returned logging events from Seq.
type Events []struct {
	Timestamp             time.Time `json:"Timestamp"`
	MessageTemplateTokens []struct {
		Text string `json:"Text"`
	} `json:"MessageTemplateTokens"`
	Properties []struct {
		Name  string `json:"Name"`
		Value any    `json:"Value"`
	} `json:"Properties"`
	ID string `json:"Id"`
}

// GetEvents takes returns the last n logging events that were received by Seq.
// There is a retry mechanism implemented; `GetEvents` will keep fetching every 2 seconds, for a maximum
// of `maxRetries` times, until Jaeger returns `expectedEvents` number of events.
func (s *Seq) GetEvents(expectedEvents int, maxRetries int) (events Events, err error) {
	var resp *http.Response
	var body []byte
	endpoint := fmt.Sprintf("http://localhost:%d/api/events?count=%d", s.Ports[80].Int(), expectedEvents)

	for range maxRetries {
		resp, err = http.Get(endpoint)
		if err != nil {
			err = fmt.Errorf("must be able to get events from seq: %v", err)
			return
		}

		if resp.StatusCode != 200 {
			err = fmt.Errorf("must be a 200 response: %v", err)
			return
		}

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			err = fmt.Errorf("must be able to get set body from seq response: %v", err)
			return
		}

		err = json.Unmarshal(body, &events)
		if err != nil {
			err = fmt.Errorf("must be able unmarshal events: %v", err)
			return
		}
		if len(events) == expectedEvents {
			break
		}

		time.Sleep(time.Second * 2)
	}

	return
}

// Start starts the Seq container.
func (s *Seq) Start(ctx context.Context) (func(context.Context) error, error) {
	emptyFunc := func(context.Context) error { return nil }
	var err error

	s.Ports = make(map[int]nat.Port)

	if s.Network == nil {
		s.Network, err = network.New(ctx)
		if err != nil {
			return emptyFunc, fmt.Errorf("could not create network: %v", err)
		}
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "datalust/seq:2024.3",
			ExposedPorts: []string{"80/tcp", "5341/tcp"},
			Networks:     []string{s.Network.Name},
			WaitingFor:   wait.ForListeningPort("80/tcp"),
			Env:          map[string]string{"ACCEPT_EULA": "Y"},
		},
		Started: true,
	})
	if err != nil {
		return emptyFunc, fmt.Errorf("seq could not start: %v", err)
	}

	s.Name, err = container.Name(ctx)
	if err != nil {
		return emptyFunc, fmt.Errorf("could not get container name: %v", err)
	}
	s.Name = s.Name[1:]

	for _, portNum := range []int{80, 5341} {
		s.Ports[portNum], err = container.MappedPort(ctx, nat.Port(fmt.Sprintf("%d", portNum)))
		if err != nil {
			return emptyFunc, fmt.Errorf("the port %d could not be retrieved: %v", portNum, err)
		}
	}

	return func(ctx context.Context) error {
		return container.Terminate(ctx, testcontainers.StopTimeout(time.Second*30))
	}, nil
}
