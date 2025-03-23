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
	"github.com/adreasnow/otelstack/seq"

	"github.com/testcontainers/testcontainers-go/network"
)

// Stack holds structs containing to all the testcontainers.
type Stack struct {
	Collector collector.Collector
	Jaeger    jaeger.Jaeger
	Seq       seq.Seq
}

// New creates a new Stack and popultes it with child container structs.
func New() *Stack {
	s := new(Stack)
	s.Collector = collector.Collector{}
	s.Jaeger = jaeger.Jaeger{}
	s.Seq = seq.Seq{}

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

// Start creates a testcotainer network and starts up all the child containers.
func (s *Stack) Start(ctx context.Context) (func(context.Context) error, error) {
	emptyFunc := func(context.Context) error { return nil }
	network, err := network.New(ctx)
	if err != nil {
		return emptyFunc, fmt.Errorf("could not create network: %v", err)
	}

	s.Jaeger.Network = network
	jaegerShutdownFunc, err := s.Jaeger.Start(ctx)
	if err != nil {
		return emptyFunc, fmt.Errorf("could not start jaeger: %v", err)
	}

	s.Seq.Network = network
	seqShutdownFunc, err := s.Seq.Start(ctx)
	if err != nil {
		if err := jaegerShutdownFunc(ctx); err != nil {
			fmt.Printf("could not shut down jaeger container: %v", err)
		}
		return emptyFunc, fmt.Errorf("could not start seq: %v", err)
	}

	s.Collector.Network = network
	collectorShutdownFunc, err := s.Collector.Start(ctx, s.Jaeger.Name, s.Seq.Name)
	if err != nil {
		if err := jaegerShutdownFunc(ctx); err != nil {
			fmt.Printf("could not shut down jaeger container: %v", err)
		}
		if err := seqShutdownFunc(ctx); err != nil {
			fmt.Printf("could not shut down seq container: %v", err)
		}
		return emptyFunc, fmt.Errorf("could not start collector: %v", err)
	}

	shutdownFunc := func(context.Context) error {
		err1 := jaegerShutdownFunc(ctx)
		err2 := seqShutdownFunc(ctx)
		err3 := collectorShutdownFunc(ctx)
		err4 := network.Remove(ctx)
		return errors.Join(err1, err2, err3, err4)
	}

	return shutdownFunc, nil
}
