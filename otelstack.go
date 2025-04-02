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
}

// New creates a new Stack and populates it with child container structs.
func New() *Stack {
	s := new(Stack)
	s.Collector = collector.Collector{}
	s.Jaeger = jaeger.Jaeger{}
	s.Seq = seq.Seq{}
	s.Prometheus = prometheus.Prometheus{}

	return s
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
	emptyFunc := func(context.Context) error { return nil }
	network, err := network.New(ctx)
	if err != nil {
		return emptyFunc, fmt.Errorf("could not create network: %v", err)
	}

	s.Jaeger.Network = network
	jaegerShutdown, err := s.Jaeger.Start(ctx)
	if err != nil {
		if err := network.Remove(ctx); err != nil {
			fmt.Printf("could not shut down network: %v", err)
		}
		return emptyFunc, fmt.Errorf("could not start jaeger: %v", err)
	}

	s.Seq.Network = network
	seqShutdown, err := s.Seq.Start(ctx)
	if err != nil {
		if err := network.Remove(ctx); err != nil {
			fmt.Printf("could not shut down network: %v", err)
		}
		if err := jaegerShutdown(context.Background()); err != nil {
			fmt.Printf("could not shut down jaeger container: %v", err)
		}
		return emptyFunc, fmt.Errorf("could not start seq: %v", err)
	}

	s.Collector.Network = network
	collectorShutdown, err := s.Collector.Start(ctx, s.Jaeger.Name, s.Seq.Name)
	if err != nil {
		if err := network.Remove(ctx); err != nil {
			fmt.Printf("could not shut down network: %v", err)
		}
		if err := jaegerShutdown(context.Background()); err != nil {
			fmt.Printf("could not shut down jaeger container: %v", err)
		}
		if err := seqShutdown(context.Background()); err != nil {
			fmt.Printf("could not shut down seq container: %v", err)
		}
		return emptyFunc, fmt.Errorf("could not start collector: %v", err)
	}

	s.Prometheus.Network = network
	prometheusShutdown, err := s.Prometheus.Start(ctx, s.Collector.Name)
	if err != nil {
		if err := network.Remove(ctx); err != nil {
			fmt.Printf("could not shut down network: %v", err)
		}
		if err := jaegerShutdown(context.Background()); err != nil {
			fmt.Printf("could not shut down jaeger container: %v", err)
		}
		if err := seqShutdown(context.Background()); err != nil {
			fmt.Printf("could not shut down seq container: %v", err)
		}
		if err := collectorShutdown(context.Background()); err != nil {
			fmt.Printf("could not shut down collector container: %v", err)
		}
		return emptyFunc, fmt.Errorf("could not start prometheus: %v", err)
	}

	shutdownFunc := func(context.Context) error {
		err1 := jaegerShutdown(context.Background())
		err2 := seqShutdown(context.Background())
		err3 := collectorShutdown(context.Background())
		err4 := prometheusShutdown(context.Background())
		err5 := network.Remove(ctx)
		return errors.Join(err1, err2, err3, err4, err5)
	}

	return shutdownFunc, nil
}
