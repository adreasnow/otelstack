// Package prometheus holds the resources needed to start a Prometheus testcontainer container.
package prometheus

import (
	"errors"
	"fmt"
	"time"

	"github.com/adreasnow/otelstack/request"
	"github.com/google/go-querystring/query"
)

type unmarshalStruct struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string    `json:"resultType"`
		Result     []Metrics `json:"result"`
	} `json:"data"`
}

// Metrics represents a Prometheus metric series.
type Metrics struct {
	Metric map[string]string `json:"metric"`
	Values [][]any           `json:"values"`
}

type requestStruct struct {
	Query string `url:"query"`
	Start string `url:"start,omitempty"`
	End   string `url:"end,omitempty"`
	Step  string `url:"step,omitempty"`
}

var errRespCode = fmt.Errorf("the return was not of status 200")

// GetMetrics takes in a service names and returns the last n `metricName` events corresponding to that `service` over that `since`.
// There is a retry mechanism implemented; `GetMetrics` will keep fetching every 2 seconds, for a maximum
// of `maxRetries` times, until Prometheus returns `expectedDataPoints` number of metrics points.
func (p *Prometheus) GetMetrics(expectedDataPoints int, maxRetries int, metricName string, service string, since time.Duration) (Metrics, string, error) {
	var endpoint string
	var metrics Metrics
	startTime := time.Now()

	var attempts int
	for {
		attempts++
		if attempts > 1 {
			time.Sleep(time.Second * 2)
		}

		sinceStart := time.Since(startTime)
		r := requestStruct{
			Query: fmt.Sprintf("%s{service_name=\"%s\"}", metricName, service),
			Start: time.Now().Add(-since - sinceStart).Format(time.RFC3339),
			End:   time.Now().Format(time.RFC3339),
			Step:  "10s",
		}

		v, queryErr := query.Values(r)
		if queryErr != nil {
			return metrics, "", fmt.Errorf("prometheus: could not marshal values into a url query for request %v: %w", r, queryErr)
		}

		endpoint = fmt.Sprintf("http://localhost:%d/api/v1/query_range?%s", p.Ports[9090].Int(), v.Encode())

		var u unmarshalStruct
		err := request.Request(endpoint, &u)
		if err != nil && !errors.Is(err, errRespCode) {
			return metrics, endpoint, fmt.Errorf("prometheus: request returned a non-retryable error: %w", err)
		}

		if len(u.Data.Result) > 0 {
			metrics = u.Data.Result[0]
		}

		if len(metrics.Values) >= expectedDataPoints {
			return metrics, endpoint, nil
		}

		if attempts >= maxRetries {
			return metrics, endpoint, fmt.Errorf("prometheus: could not get %d metrics in %d attempts", expectedDataPoints, maxRetries)
		}
	}
}
