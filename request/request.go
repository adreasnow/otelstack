// Package request provides functions for making HTTP requests and unmarshaling the response body into a struct.
package request

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
)

// ErrRetryableCode is returned when the response status code is not 200 and is retryable.
var ErrRetryableCode = fmt.Errorf("the return was not of status 200")

// ErrNonRetryableCode is returned when the response status code is not 200 and is not retryable.
var ErrNonRetryableCode = fmt.Errorf("the return was not of status 200")

var retryCodes = []int{
	http.StatusRequestTimeout,
	http.StatusTooManyRequests,
	http.StatusInternalServerError,
	http.StatusBadGateway,
	http.StatusServiceUnavailable,
	http.StatusGatewayTimeout,
	http.StatusConflict,
	http.StatusLocked,
	http.StatusTooEarly,
}

// Request sends a GET request to the specified endpoint and unmarshals the response body into the provided struct.
func Request[U any](endpoint string, unmarshal *U) (err error) {
	resp, err := http.Get(endpoint)
	if err != nil {
		return fmt.Errorf("request: could not get response on endpoint %s: %w", endpoint, err)
	}

	defer func() {
		if deferErr := resp.Body.Close(); deferErr != nil {
			err = fmt.Errorf("request: error while closing response body %s: %w", endpoint, deferErr)
		}
	}()

	if resp.StatusCode != 200 {
		var err error
		switch slices.Contains(retryCodes, resp.StatusCode) {
		case true:
			err = ErrRetryableCode
		case false:
			err = ErrNonRetryableCode
		}

		return fmt.Errorf("request: response from was not 200: got %d on endpoint %s: %w", resp.StatusCode, endpoint, err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("request: could not read body from response for endpoint %s: %w", endpoint, err)
	}

	err = json.Unmarshal(body, unmarshal)
	if err != nil {
		return fmt.Errorf("request: could not unmarshal response body %s: %w", string(body), err)
	}

	return nil
}
