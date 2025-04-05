// Package jaeger holds the resources needed to start a Jaeger testcontainer container.
package jaeger

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/adreasnow/otelstack/request"
)

type unmarshalStruct struct {
	Traces Traces `json:"data"`
	Total  int    `json:"total"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
	Errors any    `json:"errors"`
}

// Traces holds the returned traces from Jaeger.
type Traces []struct {
	TraceID   string `json:"traceID"`
	Spans     []Span `json:"spans"`
	Processes struct {
		P1 struct {
			ServiceName string `json:"serviceName"`
			Tags        []any  `json:"tags"`
		} `json:"p1"`
	} `json:"processes"`
	Warnings any `json:"warnings"`
}

// Span holds the data for each span in a trace
type Span struct {
	TraceID       string      `json:"traceID"`
	SpanID        string      `json:"spanID"`
	OperationName string      `json:"operationName"`
	References    []Reference `json:"references"`
	StartTime     int64       `json:"startTime"`
	Duration      int         `json:"duration"`
	Tags          []KeyValue  `json:"tags"`
	Logs          []Log       `json:"logs"`
	ProcessID     string      `json:"processID"`
	Warnings      any         `json:"warnings"`
}

// KeyValue holds the key-value store of data within a span
type KeyValue struct {
	Key   string `json:"key"`
	Type  string `json:"type"`
	Value any    `json:"value"`
}

// Reference holds the the relationship data between spans
type Reference struct {
	RefType string `json:"refType"`
	TraceID string `json:"traceID"`
	SpanID  string `json:"spanID"`
}

// Log holds the data for a log event
type Log struct {
	Timestamp int64      `json:"timestamp"`
	Fields    []KeyValue `json:"fields"`
}

var errRespCode = fmt.Errorf("the return was not of status 200")

// GetTraces takes in a service names and returns the last n traces corresponding to that service.
// There is a retry mechanism implemented; `GetTraces` will keep fetching every 2 seconds, for a maximum
// of `maxRetries` times, until Jaeger returns `expectedTraces` number of traces.
func (j *Jaeger) GetTraces(expectedTraces int, maxRetries int, service string) (Traces, string, error) {
	var traces Traces
	endpoint := fmt.Sprintf("http://localhost:%d/api/traces?service=%s&limit=%d", j.Ports[16686].Int(), url.QueryEscape(service), expectedTraces)

	var attempts int
	for {
		attempts++
		if attempts > 1 {
			time.Sleep(time.Second * 2)
		}

		var u unmarshalStruct
		err := request.Request(endpoint, &u)
		if err != nil && !errors.Is(err, errRespCode) {
			return traces, endpoint, fmt.Errorf("jaeger: request returned a non-retryable error: %w", err)
		}

		traces = u.Traces

		if len(traces) >= expectedTraces {
			return traces, endpoint, nil
		}

		if attempts >= maxRetries {
			return traces, endpoint, fmt.Errorf("jaeger: could not get %d traces in %d attempts", expectedTraces, maxRetries)
		}
	}
}
