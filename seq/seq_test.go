package seq

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeqStart(t *testing.T) {
	t.Parallel()
	s := Seq{}
	shutdownFunc, err := s.Start(t.Context())
	require.NoError(t, err, "seq must be able to start")
	t.Cleanup(func() {
		if err := shutdownFunc(t.Context()); err != nil {
			t.Logf("error shutting down seq: %v", err)
		}
	})

	time.Sleep(time.Second * 2)

	endpoint := fmt.Sprintf("http://localhost:%d", s.Ports[80].Int())

	resp, err := http.Get(endpoint)
	require.NoError(t, err, "must be able to call seq")
	assert.Equal(t, 200, resp.StatusCode, "request should be 200")
}
