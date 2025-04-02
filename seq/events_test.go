package seq

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEvents(t *testing.T) {
	t.Parallel()
	s := Seq{}
	shutdownFunc, err := s.Start(t.Context())
	require.NoError(t, err, "seq must be able to start")
	t.Cleanup(func() {
		if err := shutdownFunc(t.Context()); err != nil {
			t.Logf("error shutting down seq: %v", err)
		}
	})

	event := fmt.Appendf([]byte{}, `{
    "Events": [
      {
        "Timestamp": "%s",
        "MessageTemplate": "Test Message",
        "Level": "Information"
      }
    ]
  }`, time.Now().Format(time.RFC3339))

	endpoint := fmt.Sprintf("http://localhost:%d/api/events/raw", s.Ports[80].Int())

	req, err := http.NewRequestWithContext(t.Context(), "POST", endpoint, bytes.NewBuffer(event))
	req.Header.Set("Content-Type", "application/json")
	require.NoError(t, err, "request must be generated")

	time.Sleep(time.Second * 3)

	client := &http.Client{}
	resp, err := client.Do(req)

	require.NoError(t, err, "must be able to send request to seq")
	require.Equal(t, 201, resp.StatusCode, "request must be created")

	time.Sleep(time.Second * 4)

	events, err := s.GetEvents(1, 30)
	require.NoError(t, err, "must be able to get events")
	require.Len(t, events, 1)
	require.Len(t, events[0].MessageTemplateTokens, 1)
	assert.Equal(t, "Test Message", events[0].MessageTemplateTokens[0].Text)
}
