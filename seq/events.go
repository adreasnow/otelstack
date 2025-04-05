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
			err = errors.Wrapf(err, "seq: could not get log event from seq on endpoint %s", endpoint)
			return
		}

		if resp.StatusCode != 200 {
			err = errors.Wrapf(err, "seq: response from seq was not 200: got %d on endpoint %s", resp.StatusCode, endpoint)
			return
		}

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			err = errors.Wrapf(err, "seq: could not read body from seq response for endpoint %s", endpoint)
			return
		}

		err = json.Unmarshal(body, &events)
		if err != nil {
			err = errors.Wrapf(err, "seq: could not unmarshal response into events for body %s", string(body))
			return
		}
		if len(events) == expectedEvents {
			break
		}

		time.Sleep(time.Second * 2)
	}

	if len(events) < expectedEvents {
		err = errors.Wrapf(err, "seq: could not get %d events in %d attempts", expectedEvents, maxRetries)
	}

	return
}
