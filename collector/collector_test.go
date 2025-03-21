package collector

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateConfig(t *testing.T) {
	t.Parallel()
	c := Collector{}
	c.generateConfig("jaeger", "seq")

	assert.Contains(t, c.config, "endpoint: http://seq/ingest/otlp")
	assert.Contains(t, c.config, "endpoint: jaeger:4317")

}

func TestCollectorStart(t *testing.T) {
	t.Parallel()
	c := Collector{}
	shutdownFunc, err := c.Start(t.Context(), "999", "888")
	require.NoError(t, err, "collector must be able to start")
	t.Cleanup(func() {
		if err := shutdownFunc(t.Context()); err != nil {
			// do nothing
		}
	})
	endpoint := fmt.Sprintf("http://localhost:%d/health/status", c.Ports[13133].Int())

	resp, err := http.Get(endpoint)

	time.Sleep(time.Minute * 1)

	require.NoError(t, err, "must be able to call collector")
	assert.Equal(t, 200, resp.StatusCode, "request should be 200")
}
