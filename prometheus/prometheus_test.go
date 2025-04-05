package prometheus

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
	p := Prometheus{}
	p.generateConfig("collector")

	assert.Contains(t, p.config, `- targets: ["collector:8889"]`)
}

func TestPrometheusStart(t *testing.T) {
	t.Parallel()
	p := Prometheus{}
	shutdownFunc, err := p.Start(t.Context(), "test")
	require.NoError(t, err, "prometheus must be able to start")
	t.Cleanup(func() {
		if err := shutdownFunc(t.Context()); err != nil {
			t.Logf("error shutting down prometheus: %v", err)
		}
	})

	time.Sleep(time.Second*2)

	endpoint := fmt.Sprintf("http://localhost:%d", p.Ports[9090].Int())
	t.Logf("using endpoint: %s", endpoint)

	resp, err := http.Get(endpoint)
	require.NoError(t, err, "must be able to call prometheus")
	assert.Equal(t, 200, resp.StatusCode, "request should be 200")
}
