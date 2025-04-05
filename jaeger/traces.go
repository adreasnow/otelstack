// Package jaeger holds the resources needed to start a Jaeger testcontainer container.
package jaeger

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
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
			err = errors.Wrapf(err, "jaeger: could not get traces from jaeger on endpoint %s", endpoint)
			return
		}

		if resp.StatusCode != 200 {
			err = errors.Wrapf(err, "jaeger: response from jaeger was not 200: got %d on endpoint %s", resp.StatusCode, endpoint)
			return
		}

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			err = errors.Wrapf(err, "jaeger: could not read body from jaeger response for endpoint %s", endpoint)
			return
		}

		var u unmarshalStruct
		err = json.Unmarshal(body, &u)
		if err != nil {
			err = errors.Wrapf(err, "jaeger: could not unmarshal response into traces for body %s", string(body))
			return
		}

		traces = u.Traces

		if len(traces) == expectedTraces {
			break
		}

		time.Sleep(time.Second * 2)
	}

	if len(traces) < expectedTraces {
		err = errors.Wrapf(err, "jaeger: could not get %d traces in %d attempts", expectedTraces, maxRetries)
	}

	return
}
