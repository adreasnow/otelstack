package jaeger

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJaegerStart(t *testing.T) {
	t.Parallel()
	j := Jaeger{}
	shutdownFunc, err := j.Start(t.Context())
	require.NoError(t, err, "jaeger must be able to start")
	t.Cleanup(func() {
		if err := shutdownFunc(context.Background()); err != nil {
			t.Logf("error shutting down jaeger: %v", err)
		}
	})

	endpoint := fmt.Sprintf("http://localhost:%d", j.Ports[16686].Int())
	t.Logf("using endpoint: %s", endpoint)

	resp, err := http.Get(endpoint)
	require.NoError(t, err, "must be able to call jaeger")
	assert.Equal(t, 200, resp.StatusCode, "request should be 200")
}
