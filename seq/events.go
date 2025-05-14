// Package seq holds the resources needed to start a Seq testcontainer.
package seq

import (
	"errors"
	"fmt"
	"time"

	"github.com/adreasnow/otelstack/request"
)

// Events holds the returned logging events from Seq.
type Events []struct {
	Timestamp  time.Time  `json:"Timestamp"`
	Properties []Property `json:"Properties"`
	Messages   []Message  `json:"MessageTemplateTokens"`
	EventType  string     `json:"EventType"`
	Exception  string     `json:"Exception"`
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

var errRespCode = fmt.Errorf("the return was not of status 200")

// GetEvents takes returns the last n logging events that were received by Seq.
// There is a retry mechanism implemented; `GetEvents` will keep fetching every 2 seconds, for a maximum
// of `maxRetries` times, until Jaeger returns `expectedEvents` number of events.
func (s *Seq) GetEvents(expectedEvents int, maxRetries int) (Events, string, error) {
	var events Events
	endpoint := fmt.Sprintf("http://localhost:%d/api/events?count=%d", s.Ports[80].Int(), expectedEvents)

	var attempts int
	for {
		attempts++
		if attempts > 1 {
			time.Sleep(time.Second * 2)
		}

		err := request.Request(endpoint, &events)
		if err != nil && !errors.Is(err, errRespCode) {
			return events, endpoint, fmt.Errorf("seq: request returned a non-retryable error: %w", err)
		}

		if len(events) >= expectedEvents {
			return events, endpoint, nil
		}

		if attempts >= maxRetries {
			return events, endpoint, fmt.Errorf("seq: could not get %d events in %d attempts", expectedEvents, maxRetries)
		}
	}
}
