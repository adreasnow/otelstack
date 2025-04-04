// Package otelstack provides a full OTEL collector and receiver clients
// conveniently contained within testcontainers. It removes the hassle
// of managing inter-container communication, has built in querying
// for validating your tests, and uses lightweight services (seq and Jaeger) to keep
// start time low.
package otelstack

import (
	"context"
	stderr "errors"
	"fmt"
	"slices"
	"testing"

	"github.com/adreasnow/otelstack/collector"
	"github.com/adreasnow/otelstack/jaeger"
	"github.com/adreasnow/otelstack/prometheus"
	"github.com/adreasnow/otelstack/seq"

	"github.com/pkg/errors"
	"github.com/testcontainers/testcontainers-go/network"
)

// Stack holds structs containing to all the testcontainers.
type Stack struct {
	Collector  collector.Collector
	Jaeger     jaeger.Jaeger
	Seq        seq.Seq
	Prometheus prometheus.Prometheus
	metrics    bool
	logs       bool
	traces     bool
}

// New creates a new Stack and populates it with child container structs.
// Setting the services toggles will disables or enable the respective receiver containers.
func New(metrics bool, logs bool, traces bool) *Stack {
	return &Stack{
		Collector:  collector.Collector{},
		Jaeger:     jaeger.Jaeger{},
		Seq:        seq.Seq{},
		Prometheus: prometheus.Prometheus{},
		metrics:    metrics,
		logs:       logs,
		traces:     traces,
	}
}

// SetTestEnvGRPC sets the environment variableOTEL_EXPORTER_OTLP_ENDPOINT
// to the gRPC endpoint.
func (s *Stack) SetTestEnvGRPC(t *testing.T) {
	endpoint := fmt.Sprintf("http://localhost:%d", s.Collector.Ports[4317].Int())
	t.Logf(" setting endpoint to %s", endpoint)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", endpoint)
}

// SetTestEnvHTTP sets the environment variableOTEL_EXPORTER_OTLP_ENDPOINT
// to the HTTP endpoint
func (s *Stack) SetTestEnvHTTP(t *testing.T) {
	endpoint := fmt.Sprintf("http://localhost:%d", s.Collector.Ports[4318].Int())
	t.Logf(" setting endpoint to %s", endpoint)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", endpoint)
}

// Start creates a testcontainer network and starts up all the child containers.
func (s *Stack) Start(ctx context.Context) (func(context.Context) error, error) {
	shutdownFuncs := []func(context.Context) error{}
	emptyFunc := func(context.Context) error { return nil }

	shutdown := func(ctx context.Context) (err error) {
		// Reverse the slice so that the network is shut down last
		slices.Reverse(shutdownFuncs)
		for _, f := range shutdownFuncs {
			err = stderr.Join(err, f(ctx))
			err = errors.WithMessage(err, "otelstack: error shutting down container")
		}
		return
	}

	network, err := network.New(ctx)
	if err != nil {
		return shutdown, errors.WithMessage(err, "otelstack: could not create new network")
	}
	shutdownFuncs = append(shutdownFuncs, network.Remove)

	if s.traces {
		s.Jaeger.Network = network
		jaegerShutdown, err := s.Jaeger.Start(ctx)
		if err != nil {
			err = errors.WithMessage(err, "otelstack: could not start jaeger")
			return emptyFunc, stderr.Join(
				err, errors.WithMessage(shutdown(ctx), "otelstack: error occurred while shutting down services after failed jaeger start"),
			)
		}
		shutdownFuncs = append(shutdownFuncs, jaegerShutdown)
	}

	if s.logs {
		s.Seq.Network = network
		seqShutdown, err := s.Seq.Start(ctx)
		if err != nil {
			err = errors.WithMessage(err, "otelstack: could not start seq")
			return emptyFunc, stderr.Join(
				err, errors.WithMessage(shutdown(ctx), "otelstack: error occurred while shutting down services after failed seq start"),
			)
		}
		shutdownFuncs = append(shutdownFuncs, seqShutdown)
	}

	s.Collector.Network = network
	collectorShutdown, err := s.Collector.Start(ctx, s.Jaeger.Name, s.Seq.Name)
	if err != nil {
		err = errors.WithMessage(err, "otelstack: could not start collector")
		return emptyFunc, stderr.Join(
			err, errors.WithMessage(shutdown(ctx), "otelstack: error occurred while shutting down services after failed collector start"),
		)
	}
	shutdownFuncs = append(shutdownFuncs, collectorShutdown)

	if s.metrics {
		s.Prometheus.Network = network
		prometheusShutdown, err := s.Prometheus.Start(ctx, s.Collector.Name)
		if err != nil {
			err = errors.WithMessage(err, "otelstack: could not start prometheus")
			return emptyFunc, stderr.Join(
				err, errors.WithMessage(shutdown(ctx), "otelstack: error occurred while shutting down services after failed prometheus start"),
			)
		}
		shutdownFuncs = append(shutdownFuncs, prometheusShutdown)
	}

	return shutdown, nil
}
