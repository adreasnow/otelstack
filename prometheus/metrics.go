// Package prometheus holds the resources needed to start a Prometheus testcontainer container.
package prometheus

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/pkg/errors"
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
	Metric struct {
		Name                  string `json:"__name__"`
		DeploymentEnvironment string `json:"deployment_environment"`
		ExportedInstance      string `json:"exported_instance"`
		ExportedJob           string `json:"exported_job"`
		Instance              string `json:"instance"`
		Job                   string `json:"job"`
		LibraryLanguage       string `json:"library_language"`
		ServiceInstanceID     string `json:"service_instance_id"`
		ServiceName           string `json:"service_name"`
	} `json:"metric"`
	Values [][]any `json:"values"`
}

type request struct {
	Query string `url:"query"`
	Start string `url:"start,omitempty"`
	End   string `url:"end,omitempty"`
	Step  string `url:"step,omitempty"`
}

// GetMetrics takes in a service names and returns the last n `metricName` events corresponding to that `service` over that `since`.
// There is a retry mechanism implemented; `GetMetrics` will keep fetching every 2 seconds, for a maximum
// of `maxRetries` times, until Prometheus returns `expectedDataPoints` number of metrics points.
func (p *Prometheus) GetMetrics(expectedDataPoints int, maxRetries int, metricName string, service string, since time.Duration) (metrics Metrics, endpoint string, err error) {
	var resp *http.Response
	var body []byte
	startTime := time.Now()

	for range maxRetries {
		sinceStart := time.Since(startTime)
		request := request{
			Query: fmt.Sprintf("%s{service_name=\"%s\"}", metricName, service),
			Start: time.Now().Add(-since - sinceStart).Format(time.RFC3339),
			End:   time.Now().Format(time.RFC3339),
			Step:  "10s",
		}

		v, queryErr := query.Values(request)
		if queryErr != nil {
			err = errors.Wrapf(queryErr, "prometheus: could not marshal values into a url query for request %v", request)
			return
		}

		endpoint = fmt.Sprintf("http://localhost:%d/api/v1/query_range?%s", p.Ports[9090].Int(), v.Encode())

		resp, err = http.Get(endpoint)
		if err != nil {
			err = errors.Wrapf(err, "prometheus: could not get metrics from prometheus on endpoint %s", endpoint)
			return
		}

		if resp.StatusCode != 200 {
			err = fmt.Errorf("prometheus: response from prometheus was not 200: got %d on endpoint %s", resp.StatusCode, endpoint)
			return
		}

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			err = errors.Wrapf(err, "prometheus: could not read body from prometheus response for endpoint %s", endpoint)
			return
		}

		var u unmarshalStruct
		err = json.Unmarshal(body, &u)
		if err != nil {
			err = errors.Wrapf(err, "prometheus: could not unmarshal response into metrics for body %s", string(body))
			return
		}

		if len(u.Data.Result) > 0 {
			metrics = u.Data.Result[0]

			if len(metrics.Values) >= expectedDataPoints {
				break
			}
		}

		time.Sleep(time.Second * 2)
	}

	if len(metrics.Values) < expectedDataPoints {
		err = errors.Wrapf(err, "prometheus: could not get %d metric points in %d attempts", expectedDataPoints, maxRetries)
	}

	return
}
