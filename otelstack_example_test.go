package otelstack

import (
	"context"
	"testing"

	"github.com/adreasnow/otelstack/jaeger"
	"github.com/adreasnow/otelstack/seq"
	"github.com/stretchr/testify/require"
)

func TestSetupStack(t *testing.T) {
	stack := New()
	shutdownFunc, err := stack.Start(t.Context())
	require.NoError(t, err, "the stack must start up")

	// be sure to defer shutdown of the stack
	t.Cleanup(func() {
		if err := shutdownFunc(context.Background()); err != nil {
			// do nothing
		}
	})

	// For optionally setting OTEL_EXPORTER_OTLP_INSECURE and OTEL_EXPORTER_OTLP_ENDPOINT
	stack.SetTestEnv(t)

	// ports can be accessed as such
	t.Logf("Seq ui: http://localhost:%d", stack.Seq.Ports[80].Int())
	t.Logf("Jaeger ui: http://localhost:%d", stack.Seq.Ports[16686].Int())

	// Continue to initialise your own otel setup here

	// Your telemetry will now be sent to the stack
}

// Containers can also be started independently if needed
func TestSetupSeq(t *testing.T) {
	seq := seq.Seq{}
	shutdownFunc, err := seq.Start(t.Context())
	require.NoError(t, err, "the container must start up")

	// be sure to defer shutdown of the stack
	t.Cleanup(func() {
		if err := shutdownFunc(context.Background()); err != nil {
			// do nothing
		}
	})

	t.Logf("Seq ui: http://localhost:%d", seq.Ports[80].Int())
}

func TestSetupJaeger(t *testing.T) {
	jaeger := jaeger.Jaeger{}
	shutdownFunc, err := jaeger.Start(t.Context())
	require.NoError(t, err, "the container must start up")

	// be sure to defer shutdown of the stack
	t.Cleanup(func() {
		if err := shutdownFunc(context.Background()); err != nil {
			// do nothing
		}
	})

	t.Logf("Jaeger ui: http://localhost:%d", jaeger.Ports[16686].Int())
}
