// Package seq holds the resources needed to start a Seq testcontainer.
package seq

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// Events holds the returned logging events from Seq.
type Events []struct {
	Timestamp  time.Time  `json:"Timestamp"`
	Properties []Property `json:"Properties"`
	Messages   []Message  `json:"MessageTemplateTokens"`
	EventType  string     `json:"EventType"`
	Level      string     `json:"Level"`
	TraceID    string     `json:"TraceId"`
	SpanID     string     `json:"SpanId"`
	SpanKind   string     `json:"SpanKind"`
	Resource   []Resource `json:"Resource"`
	ID         string     `json:"Id"`
	Links      struct {
		Self  string `json:"Self"`
		Group string `json:"Group"`
	} `json:"Links"`
}

// Message holds the message template tokens from Seq.
type Message struct {
	Text string `json:"Text"`
}

// Property holds the property name and value from Seq.
type Property struct {
	Name  string `json:"Name"`
	Value any    `json:"Value"`
}

// Resource holds the resource name and value from Seq.
type Resource struct {
	Name  string `json:"Name"`
	Value struct {
		Name string `json:"name"`
	} `json:"Value"`
}

// GetEvents takes returns the last n logging events that were received by Seq.
// There is a retry mechanism implemented; `GetEvents` will keep fetching every 2 seconds, for a maximum
// of `maxRetries` times, until Jaeger returns `expectedEvents` number of events.
func (s *Seq) GetEvents(expectedEvents int, maxRetries int) (Events, string, error) {
	var events Events

	endpoint := fmt.Sprintf("http://localhost:%d/api/events?count=%d", s.Ports[80].Int(), expectedEvents)

	for range maxRetries {
		resp, err := http.Get(endpoint)
		if err != nil {
			return events, endpoint, errors.Wrapf(err, "seq: could not get log event from seq on endpoint %s", endpoint)
		}

		if resp.StatusCode != 200 {
			return events, endpoint, fmt.Errorf("seq: response from seq was not 200: got %d on endpoint %s", resp.StatusCode, endpoint)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return events, endpoint, errors.Wrapf(err, "seq: could not read body from seq response for endpoint %s", endpoint)
		}

		err = json.Unmarshal(body, &events)
		if err != nil {
			return events, endpoint, errors.Wrapf(err, "seq: could not unmarshal response into events for body %s", string(body))
		}

		if len(events) == expectedEvents {
			break
		}

		time.Sleep(time.Second * 2)
	}

	if len(events) < expectedEvents {
		return events, endpoint, fmt.Errorf("seq: could not get %d events in %d attempts", expectedEvents, maxRetries)
	}

	return events, endpoint, nil
}
