// Package otelstack provides a full OTEL collector and receiver clients
// conveniently contained within testcontainers. It removes the hassle
// of managing inter-container communication, has built in querying
// for validating your tests, and uses lightweight services (seq and Jaeger) to keep
// start time low.
package otelstack

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/adreasnow/otelstack/collector"
	"github.com/adreasnow/otelstack/jaeger"
	"github.com/adreasnow/otelstack/prometheus"
	"github.com/adreasnow/otelstack/seq"

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

	shutdown := func(ctx context.Context) error {
		// Reverse the slice so that the network is shut down last
		slices.Reverse(shutdownFuncs)
		for _, f := range shutdownFuncs {
			var err error
			err = errors.Join(err, f(ctx))
			if err != nil {
				err = fmt.Errorf("otelstack: error shutting down container: %w", err)
				return err
			}
		}
		return nil
	}

	network, err := network.New(ctx)
	if err != nil {
		return shutdown, fmt.Errorf("otelstack: could not create new network: %w", err)
	}
	shutdownFuncs = append(shutdownFuncs, network.Remove)

	if s.traces {
		s.Jaeger.Network = network
		jaegerShutdown, err := s.Jaeger.Start(ctx)
		if err != nil {
			err = fmt.Errorf("otelstack: could not start jaeger: %w", err)
			if shutdownErr := shutdown(ctx); shutdownErr != nil {
				err = errors.Join(
					err, fmt.Errorf("otelstack: error occurred while shutting down services after failed jaeger start: %w", shutdownErr),
				)
			}
			return emptyFunc, err
		}
		shutdownFuncs = append(shutdownFuncs, jaegerShutdown)
	}

	if s.logs {
		s.Seq.Network = network
		seqShutdown, err := s.Seq.Start(ctx)
		if err != nil {
			err = fmt.Errorf("otelstack: could not start seq: %w", err)
			if shutdownErr := shutdown(ctx); shutdownErr != nil {
				err = errors.Join(
					err, fmt.Errorf("otelstack: error occurred while shutting down services after failed seq start: %w", shutdownErr),
				)
			}
			return emptyFunc, err
		}
		shutdownFuncs = append(shutdownFuncs, seqShutdown)
	}

	s.Collector.Network = network
	collectorShutdown, err := s.Collector.Start(ctx, s.Jaeger.Name, s.Seq.Name)
	if err != nil {
		err = fmt.Errorf("otelstack: could not start collector: %w", err)
		if shutdownErr := shutdown(ctx); shutdownErr != nil {
			err = errors.Join(
				err, fmt.Errorf("otelstack: error occurred while shutting down services after failed collector start: %w", shutdownErr),
			)
		}
		return emptyFunc, err
	}
	shutdownFuncs = append(shutdownFuncs, collectorShutdown)

	if s.metrics {
		s.Prometheus.Network = network
		prometheusShutdown, err := s.Prometheus.Start(ctx, s.Collector.Name)
		if err != nil {
			err = fmt.Errorf("otelstack: could not start prometheus: %w", err)
			if shutdownErr := shutdown(ctx); shutdownErr != nil {
				err = errors.Join(
					err, fmt.Errorf("otelstack: error occurred while shutting down services after failed prometheus start: %w", shutdownErr),
				)
			}
			return emptyFunc, err
		}
		shutdownFuncs = append(shutdownFuncs, prometheusShutdown)
	}

	return shutdown, nil
}
