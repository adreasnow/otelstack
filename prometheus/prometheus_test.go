package prometheus

import (
	"context"
	"fmt"
	"net/http"
	"testing"

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
		if err := shutdownFunc(context.Background()); err != nil {
			t.Logf("error shutting down prometheus: %v", err)
		}
	})

	endpoint := fmt.Sprintf("http://localhost:%d", p.Ports[9090].Int())
	t.Logf("using endpoint: %s", endpoint)

	resp, err := http.Get(endpoint)
	require.NoError(t, err, "must be able to call prometheus")
	assert.Equal(t, 200, resp.StatusCode, "request should be 200")
}
