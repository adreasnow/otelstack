package otelstack

import (
	"context"
	"net/http"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"

	otelLogGlobal "go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var serviceName = "test-service"

func startGoroutineMeter(t *testing.T) {
	t.Helper()
	meter := otel.Meter(serviceName)
	_, err := meter.Int64ObservableGauge("goroutine.count",
		metric.WithUnit("goroutine"),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			o.Observe(int64(runtime.NumGoroutine()))
			return nil
		}),
	)
	require.NoError(t, err, "must be able to start goroutine meter")
}

func setupOTELHTTP(t *testing.T, metrics bool, logs bool, traces bool, port nat.Port) func() {
	resources, err := resource.New(t.Context(), resource.WithAttributes(attribute.String("service.name", serviceName)))
	require.NoError(t, err, "resources must be created")

	shutdownFuncs := []func(){}

	if metrics {
		opts := []otlpmetrichttp.Option{}
		if port.Int() != 0 {
			opts = append(opts,
				otlpmetrichttp.WithEndpoint("localhost:"+port.Port()),
				otlpmetrichttp.WithInsecure(),
			)
		}

		metricExporter, err := otlpmetrichttp.New(t.Context(), opts...)
		require.NoError(t, err, "must be able to set up exporter")
		otel.SetMeterProvider(
			sdkmetric.NewMeterProvider(
				sdkmetric.WithResource(resources),
				sdkmetric.WithReader(
					sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(time.Second)),
				),
			),
		)

		shutdownFuncs = append(shutdownFuncs, func() {
			if err := metricExporter.Shutdown(context.Background()); err != nil {
				t.Logf("failed to shutdown metric exporter: %v", err)
			}
		})
	}

	if logs {
		opts := []otlploghttp.Option{}
		if port.Int() != 0 {
			opts = append(opts,
				otlploghttp.WithEndpoint("localhost:"+port.Port()),
				otlploghttp.WithInsecure(),
			)
		}

		logExporter, err := otlploghttp.New(t.Context(), opts...)
		require.NoError(t, err, "log exporter must be started")
		otelLogGlobal.SetLoggerProvider(
			sdklog.NewLoggerProvider(
				sdklog.WithProcessor(
					sdklog.NewSimpleProcessor(logExporter),
				),
				sdklog.WithResource(resources),
			),
		)

		shutdownFuncs = append(shutdownFuncs, func() {
			if err := logExporter.Shutdown(context.Background()); err != nil {
				t.Logf("failed to shutdown logs exporter: %v", err)
			}
		})
	}

	if traces {
		opts := []otlptracehttp.Option{}
		if port.Int() != 0 {
			opts = append(opts,
				otlptracehttp.WithEndpoint("localhost:"+port.Port()),
				otlptracehttp.WithInsecure(),
			)
		}

		traceExporter, err := otlptracehttp.New(t.Context(), opts...)
		require.NoError(t, err, "trace exporter must be started")

		otel.SetTracerProvider(
			sdktrace.NewTracerProvider(
				sdktrace.WithSyncer(traceExporter),
				sdktrace.WithResource(resources),
			),
		)

		shutdownFuncs = append(shutdownFuncs, func() {
			if err := traceExporter.Shutdown(context.Background()); err != nil {
				t.Logf("failed to shutdown traces exporter: %v", err)
			}
		})
	}

	return func() {
		for _, f := range shutdownFuncs {
			f()
		}
	}
}

func setupOTELgRPC(t *testing.T, metrics bool, logs bool, traces bool, port nat.Port) func() {
	resources, err := resource.New(t.Context(), resource.WithAttributes(attribute.String("service.name", serviceName)))
	require.NoError(t, err, "resources must be created")

	shutdownFuncs := []func(){}

	if metrics {
		opts := []otlpmetricgrpc.Option{}
		if port.Int() != 0 {
			opts = append(opts,
				otlpmetricgrpc.WithEndpoint("localhost:"+port.Port()),
				otlpmetricgrpc.WithInsecure(),
			)
		}

		metricExporter, err := otlpmetricgrpc.New(t.Context(), opts...)
		require.NoError(t, err, "must be able to set up exporter")
		otel.SetMeterProvider(
			sdkmetric.NewMeterProvider(
				sdkmetric.WithResource(resources),
				sdkmetric.WithReader(
					sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(time.Second)),
				),
			),
		)

		shutdownFuncs = append(shutdownFuncs, func() {
			if err := metricExporter.Shutdown(context.Background()); err != nil {
				t.Logf("failed to shutdown metric exporter: %v", err)
			}
		})
	}

	if logs {
		opts := []otlploggrpc.Option{}
		if port.Int() != 0 {
			opts = append(opts,
				otlploggrpc.WithEndpoint("localhost:"+port.Port()),
				otlploggrpc.WithInsecure(),
			)
		}

		logExporter, err := otlploggrpc.New(t.Context(), opts...)
		require.NoError(t, err, "log exporter must be started")
		otelLogGlobal.SetLoggerProvider(
			sdklog.NewLoggerProvider(
				sdklog.WithProcessor(
					sdklog.NewSimpleProcessor(logExporter),
				),
				sdklog.WithResource(resources),
			),
		)

		shutdownFuncs = append(shutdownFuncs, func() {
			if err := logExporter.Shutdown(context.Background()); err != nil {
				t.Logf("failed to shutdown logs exporter: %v", err)
			}
		})
	}

	if traces {
		opts := []otlptracegrpc.Option{}
		if port.Int() != 0 {
			opts = append(opts,
				otlptracegrpc.WithEndpoint("localhost:"+port.Port()),
				otlptracegrpc.WithInsecure(),
			)
		}

		traceExporter, err := otlptracegrpc.New(t.Context(), opts...)
		require.NoError(t, err, "trace exporter must be started")

		otel.SetTracerProvider(
			sdktrace.NewTracerProvider(
				sdktrace.WithSyncer(traceExporter),
				sdktrace.WithResource(resources),
			),
		)

		shutdownFuncs = append(shutdownFuncs, func() {
			if err := traceExporter.Shutdown(context.Background()); err != nil {
				t.Logf("failed to shutdown traces exporter: %v", err)
			}
		})
	}

	return func() {
		for _, f := range shutdownFuncs {
			f()
		}
	}
}

func TestStart(t *testing.T) {
	t.Parallel()
	testData := []struct {
		name string
	}{{"gRPC"}, {"HTTP"}}
	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := Stack{
				logs: true,
			}
			shutdownFunc, err := s.Start(t.Context())
			require.NoError(t, err, "stack must be able to start")
			t.Cleanup(func() {
				if err := shutdownFunc(context.Background()); err != nil {
					t.Logf("error shutting down stack: %v", err)
				}
			})

			var shutdown func()
			switch tt.name {
			case "gRPC":
				shutdown = setupOTELgRPC(t, false, true, false, s.Collector.Ports[4317])
			case "HTTP":
				shutdown = setupOTELHTTP(t, false, true, false, s.Collector.Ports[4318])
			}

			t.Cleanup(shutdown)

			{ // send data
				record := log.Record{}
				record.SetTimestamp(time.Now())
				record.SetBody(log.StringValue("test message"))
				record.SetSeverity(log.SeverityError)
				record.SetSeverityText("ERROR")
				otelLogGlobal.GetLoggerProvider().
					Logger(serviceName).
					Emit(t.Context(), record)
			}

			events, _, err := s.Seq.GetEvents(1, 30)
			require.NoError(t, err)
			require.Len(t, events, 1)
			require.Len(t, events[0].Messages, 1)
			assert.Equal(t, "test message", events[0].Messages[0].Text)
		})
	}
}

func TestNew(t *testing.T) {
	t.Run("all services", func(t *testing.T) {
		t.Parallel()

		s := New(true, true, true)
		shutdownStack, err := s.Start(t.Context())
		require.NoError(t, err, "the stack must start up")

		t.Cleanup(func() {
			if err := shutdownStack(context.Background()); err != nil {
				t.Logf("error shutting down otel: %v", err)
			}
		})

		resp, err := http.Get("http://localhost:" + s.Seq.Ports[80].Port())
		require.NoError(t, err, "must be able to call seq")
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		resp, err = http.Get("http://localhost:" + s.Jaeger.Ports[16686].Port())
		require.NoError(t, err, "must be able to call jaeger")
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		resp, err = http.Get("http://localhost:" + s.Prometheus.Ports[9090].Port())
		require.NoError(t, err, "must be able to call prometheus")
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		resp, err = http.Get("http://localhost:" + s.Collector.Ports[13133].Port() + "/health/status")
		require.NoError(t, err, "must be able to call collector")
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("metrics only", func(t *testing.T) {
		t.Parallel()

		s := New(true, false, false)
		shutdownStack, err := s.Start(t.Context())
		require.NoError(t, err, "the stack must start up")

		t.Cleanup(func() {
			if err := shutdownStack(context.Background()); err != nil {
				t.Logf("error shutting down otel: %v", err)
			}
		})

		shutdownOTEL := setupOTELgRPC(t, true, false, false, s.Collector.Ports[4317])
		t.Cleanup(shutdownOTEL)

		startGoroutineMeter(t)

		metrics, _, err := s.Prometheus.GetMetrics(3, 30, "goroutine_count", serviceName, time.Second*30)
		require.NoError(t, err, "must be able to get metrics")

		require.GreaterOrEqual(t, len(metrics.Values), 3)
		require.Len(t, metrics.Values[0], 2)
		num, err := strconv.Atoi(metrics.Values[0][1].(string))
		require.NoError(t, err, "must be able to parse the metric value")

		assert.Greater(t, num, 2)
	})

	t.Run("logs only", func(t *testing.T) {
		t.Parallel()

		s := New(false, true, false)
		shutdownStack, err := s.Start(t.Context())
		require.NoError(t, err, "the stack must start up")

		t.Cleanup(func() {
			if err := shutdownStack(context.Background()); err != nil {
				t.Logf("error shutting down otel: %v", err)
			}
		})

		shutdownOTEL := setupOTELgRPC(t, false, true, false, s.Collector.Ports[4317])
		t.Cleanup(shutdownOTEL)

		{ // send data
			record := log.Record{}
			record.SetTimestamp(time.Now())
			record.SetBody(log.StringValue("test message"))
			record.SetSeverity(log.SeverityError)
			record.SetSeverityText("ERROR")
			otelLogGlobal.GetLoggerProvider().
				Logger(serviceName).
				Emit(t.Context(), record)
		}

		events, _, err := s.Seq.GetEvents(1, 30)
		require.NoError(t, err)
		require.Len(t, events, 1)
		require.Len(t, events[0].Messages, 1)
		assert.Equal(t, "test message", events[0].Messages[0].Text)
	})

	t.Run("traces only", func(t *testing.T) {
		s := New(false, false, true)
		shutdownStack, err := s.Start(t.Context())
		require.NoError(t, err, "the stack must start up")

		t.Cleanup(func() {
			if err := shutdownStack(context.Background()); err != nil {
				t.Logf("error shutting down otel: %v", err)
			}
		})

		shutdownOTEL := setupOTELgRPC(t, false, false, true, s.Collector.Ports[4317])
		t.Cleanup(shutdownOTEL)

		{ // send data
			tracer := otel.Tracer("")
			ctx, span := tracer.Start(t.Context(), "test.segment")
			trace.SpanFromContext(ctx).SetAttributes(attribute.String("test.key", "test_value"))
			time.Sleep(time.Millisecond * 100)
			span.End()
		}

		traces, _, err := s.Jaeger.GetTraces(1, 30, serviceName)

		require.NoError(t, err, "must be able to get traces")
		require.Len(t, traces, 1)
		require.Len(t, traces[0].Spans, 1)
		assert.Equal(t, "test.segment", traces[0].Spans[0].OperationName)
	})
}
