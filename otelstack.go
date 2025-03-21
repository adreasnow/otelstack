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

type stack struct {
	Collector collector.Collector
	Jaeger    jaeger.Jaeger
	Seq       seq.Seq
}

func New() *stack {
	s := new(stack)
	s.Collector = collector.Collector{}
	s.Jaeger = jaeger.Jaeger{}
	s.Seq = seq.Seq{}

	return s
}

func (s *stack) SetTestEnv(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", fmt.Sprintf("http://localhost:%d", s.Collector.Ports[4317].Int()))
}

func (s *stack) Start(ctx context.Context) (func(context.Context) error, error) {
	emptyFunc := func(context.Context) error { return nil }
	network, err := network.New(ctx)
	if err != nil {
		return emptyFunc, fmt.Errorf("could not create network: %v", err)
	}

	s.Jaeger.Network = network
	jaegerShudownFunc, err := s.Jaeger.Start(ctx)
	if err != nil {
		return emptyFunc, fmt.Errorf("could not start jaeger: %v", err)
	}

	s.Seq.Network = network
	seqShudownFunc, err := s.Seq.Start(ctx)
	if err != nil {
		if err := jaegerShudownFunc(ctx); err != nil {
			fmt.Printf("could not shut down jaeger container: %v", err)
		}
		return emptyFunc, fmt.Errorf("could not start seq: %v", err)
	}

	s.Collector.Network = network
	collectorShudownFunc, err := s.Collector.Start(ctx, s.Jaeger.Name, s.Seq.Name)
	if err != nil {
		if err := jaegerShudownFunc(ctx); err != nil {
			fmt.Printf("could not shut down jaeger container: %v", err)
		}
		if err := seqShudownFunc(ctx); err != nil {
			fmt.Printf("could not shut down seq container: %v", err)
		}
		return emptyFunc, fmt.Errorf("could not start collector: %v", err)
	}

	shutdownFunc := func(context.Context) error {
		err1 := jaegerShudownFunc(ctx)
		err2 := seqShudownFunc(ctx)
		err3 := collectorShudownFunc(ctx)
		err4 := network.Remove(ctx)
		return errors.Join(err1, err2, err3, err4)
	}

	return shutdownFunc, nil
}
