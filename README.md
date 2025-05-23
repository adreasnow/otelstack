# OTEL TestContainer stack for go

[![godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/adreasnow/otelstack) [![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/adreasnow/otelstack/main/LICENSE) [![Build Status](https://github.com/adreasnow/otelstack/actions/workflows/test.yaml/badge.svg?branch=main)](https://github.com/adreasnow/otelstack/actions/workflows/test.yaml) [![Go Coverage](https://github.com/adreasnow/otelstack/wiki/coverage.svg)](https://raw.githack.com/wiki/adreasnow/otelstack/coverage.html)

Package otelstack provides a full OTEL collector and receiver clients conveniently contained within testcontainers. It removes the hassle of managing inter-container communication, has built in querying for validating your tests, and uses lightweight services (seq, Jaeger, and Prometheus with tsdb) to keep startup time low.

## Usage

```go
stack := New()
shutdownFunc, err := stack.Start(t.Context())
require.NotNil(t, err, "the stack must start up")

// be sure to defer shutdown of the stack
t.Cleanup(func() {
  if err := shutdownFunc(context.Background()); err != nil {
    t.Logf("error shutting down stack: %w", err)
  }
})

// For optionally setting OTEL_EXPORTER_OTLP_ENDPOINT
stack.SetTestEnv(t)

// ports can be accessed as such
t.Logf("Seq ui: http://localhost:%d", stack.Seq.Ports[80].Int())
t.Logf("Jaeger ui: http://localhost:%d", stack.Seq.Ports[16686].Int())
t.Logf("Prometheus ui: http://localhost:%d", stack.Prometheus.Ports[9090].Int())

t.Logf("OTEL gRPC endpoint: http://localhost:%d", stack.Collector.Ports[4317].Int())

// Continue to initialise your own otel setup here
...

// Get traces from Jaeger
  traces, err := stack.Jaeger.GetTraces(5, serviceName)
  require.NoError(t, err, "must be able to get traces")
  assert.Equal(t, "test-segment", traces.Data[0].Spans[0].OperationName)

  // Get log events from Seq
  events, err := stack.Seq.GetEvents(5)
  require.NoError(t, err)
  assert.Equal(t, "test message", events[0].MessageTemplateTokens[0].Text)

  // Get metrics from Prometheus
  metrics, err := stack.Prometheus.GetMetrics(3, 30, "goroutine_count", serviceName, time.Second*30)
  require.NoError(t, err, "must be able to get metrics")
  assert.Greater(t, metrics.Values[0][0].(float64), 5.0)
```
