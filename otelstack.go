package otelstack

import (
	"context"
	"errors"
	"fmt"
	"otelstack/collector"
	"otelstack/jaeger"
	"otelstack/seq"

	"github.com/testcontainers/testcontainers-go/network"
)

type Stack struct {
	Collector collector.Collector
	Jaeger    jaeger.Jaeger
	Seq       seq.Seq
}

func (s *Stack) Start(ctx context.Context) (func(context.Context) error, error) {
	emptyFunc := func(context.Context) error { return nil }
	network, err := network.New(ctx)
	if err != nil {
		return emptyFunc, errors.Join(fmt.Errorf("could not create network"), err)
	}
	defer network.Remove(ctx)

	s.Jaeger = jaeger.Jaeger{Network: network}
	err, jaegerShudownFunc := s.Jaeger.Start(ctx)
	if err != nil {
		return emptyFunc, errors.Join(fmt.Errorf("could not start jaeger"), err)
	}

	s.Seq = seq.Seq{Network: network}
	err, seqShudownFunc := s.Seq.Start(ctx)
	if err != nil {
		defer jaegerShudownFunc(ctx)
		return emptyFunc, errors.Join(fmt.Errorf("could not start seq"), err)
	}

	s.Collector = collector.Collector{Network: network}
	err, collectorShudownFunc := s.Collector.Start(ctx, s.Jaeger.Name, s.Seq.Name)
	if err != nil {
		defer jaegerShudownFunc(ctx)
		defer seqShudownFunc(ctx)
		return emptyFunc, errors.Join(fmt.Errorf("could not start collector"), err)
	}

	shutdownFunc := func(context.Context) error {
		err1 := jaegerShudownFunc(ctx)
		err2 := seqShudownFunc(ctx)
		err3 := collectorShudownFunc(ctx)
		return errors.Join(err1, err2, err3)
	}

	return shutdownFunc, nil
}
