// Package jaeger holds the resources needed to start a Jaeger testcontainer container.
package jaeger

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type unmarshalStruct struct {
	Traces Traces `json:"data"`
}

// Traces holds the returned traces from Jaeger.
type Traces []struct {
	TraceID string `json:"traceID"`
	Spans   []struct {
		TraceID       string `json:"traceID"`
		SpanID        string `json:"spanID"`
		OperationName string `json:"operationName"`
	} `json:"spans"`
}

// GetTraces takes in a service names and returns the last n traces corresponding to that service.
// There is a retry mechanism implemented; `GetTraces` will keep fetching every 2 seconds, for a maximum
// of `maxRetries` times, until Jaeger returns `expectedTraces` number of traces.
func (j *Jaeger) GetTraces(expectedTraces int, maxRetries int, service string) (traces Traces, err error) {
	var resp *http.Response
	var body []byte
	endpoint := fmt.Sprintf("http://localhost:%d/api/traces?service=%s&limit=%d", j.Ports[16686].Int(), url.QueryEscape(service), expectedTraces)

	for range maxRetries {
		resp, err = http.Get(endpoint)
		if err != nil {
			err = fmt.Errorf("could not get traces from jaeger: %v", err)
			return
		}

		if resp.StatusCode != 200 {
			err = fmt.Errorf("not a 200 response, got %d", resp.StatusCode)
			return
		}

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			err = fmt.Errorf("could not get set body from seq response: %v", err)
			return
		}

		var u unmarshalStruct
		err = json.Unmarshal(body, &u)
		if err != nil {
			err = fmt.Errorf("could not unmarshal traces: %v", err)
			return
		}

		traces = u.Traces

		if len(traces) == expectedTraces {
			break
		}

		time.Sleep(time.Second * 2)
	}

	return
}
