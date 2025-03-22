# OTEL Tescontainer stack for go

[![godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/adreasnow/otelstack) [![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/adreasnow/otelstack/main/LICENSE) [![Build Status](https://github.com/adreasnow/otelstack/actions/workflows/test-tag.yaml/badge.svg?branch=main)](https://github.com/adreasnow/otelstack/actions/workflows/test-tag.yaml) [![Go Coverage](https://github.com/adreasnow/otelstack/wiki/coverage.svg)](https://raw.githack.com/wiki/adreasnow/otelstack/coverage.html)

The otelstack package provides an easy to use pop-up OTEL testcontainers stack for use in go

## Usage

```go
stack := New()
shutdownFunc, err := stack.Start(t.Context())
require.NotNil(t, err, "the stack must start up")

// be sure to defer shutdown of the stack
t.Cleanup(func() {
	if err := shutdownFunc(context.Background()); err != nil {
		t.Logf("error shutting down stack: %v", err)
	}
})

// For optionally setting OTEL_EXPORTER_OTLP_ENDPOINT
stack.SetTestEnv(t)

// ports can be accessed as such
t.Logf("Seq ui: http://localhost:%d", stack.Seq.Ports[80].Int())
t.Logf("Jaeger ui: http://localhost:%d", stack.Seq.Ports[16686].Int())

// Continue to initialise your own otel setup here
...

// Get traces from Jaeger
	traces, err := stack.Jaeger.GetTraces(t.Context(), 5, serviceName)
	require.NoError(t, err, "must be able to get traces")
	assert.Equal(t, "test-segment", traces.Data[0].Spans[0].OperationName)

	// Get log events from Seq
	events, err := stack.Seq.GetEvents(t.Context(), 5)
	require.NoError(t, err)
	assert.Equal(t, "test message", events[0].MessageTemplateTokens[0].Text)


```

## TODO

- [ ] Add http endpoint to collector
- [ ] Add metrics to stack (prometheus, but maybe grafana?)
- [ ] Increase test coverage
