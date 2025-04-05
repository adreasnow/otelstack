package seq

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeqStart(t *testing.T) {
	t.Parallel()
	s := Seq{}
	shutdownFunc, err := s.Start(t.Context())
	require.NoError(t, err, "seq must be able to start")
	t.Cleanup(func() {
		if err := shutdownFunc(context.Background()); err != nil {
			t.Logf("error shutting down seq: %v", err)
		}
	})

	endpoint := fmt.Sprintf("http://localhost:%d", s.Ports[80].Int())

	resp, err := http.Get(endpoint)
	require.NoError(t, err, "must be able to call seq")
	assert.Equal(t, 200, resp.StatusCode, "request should be 200")
}
