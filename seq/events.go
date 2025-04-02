package seq

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
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
			err = fmt.Errorf("could not get events from seq: %v", err)
			return
		}

		if resp.StatusCode != 200 {
			err = fmt.Errorf("not a 200 response: %v", err)
			return
		}

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			err = fmt.Errorf("could not get set body from seq response: %v", err)
			return
		}

		err = json.Unmarshal(body, &events)
		if err != nil {
			err = fmt.Errorf("could not unmarshal events: %v", err)
			return
		}
		if len(events) == expectedEvents {
			break
		}

		time.Sleep(time.Second * 2)
	}

	return
}
