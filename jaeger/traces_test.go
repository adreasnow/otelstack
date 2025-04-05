package jaeger

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var serviceName = "test service"

func TestGetTraces(t *testing.T) {
	t.Parallel()
	j := Jaeger{}
	shutdownFunc, err := j.Start(t.Context())
	require.NoError(t, err, "jaeger must be able to start")
	t.Cleanup(func() {
		if err := shutdownFunc(t.Context()); err != nil {
			t.Logf("error shutting down jaeger: %v", err)
		}
	})

	// set up otel tracer
	exporter, err := otlptracehttp.New(t.Context(),
		otlptracehttp.WithEndpoint("localhost:"+j.Ports[4318].Port()),
		otlptracehttp.WithInsecure(),
	)
	require.NoError(t, err, "must be able to set up exporter")

	resources, err := resource.New(t.Context(), resource.WithAttributes(attribute.String("service.name", serviceName)))
	require.NoError(t, err, "must be able to set up resources")

	otel.SetTracerProvider(
		sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithSyncer(exporter),
			sdktrace.WithResource(resources),
		),
	)

	t.Cleanup(func() {
		if err := exporter.Shutdown(context.Background()); err != nil {
			t.Logf("error shutting down exporter: %v", err)
		}
	})

	var ctx context.Context
	var span1 trace.Span
	var span2 trace.Span
	{ // send trace
		tracer := otel.Tracer(serviceName)

		func() {
			ctx, span1 = tracer.Start(ctx, "segment.parent")
			span1.AddEvent("this is a log event")
			defer span1.End()

			func() {
				// span3 has an error and log event
				_, span2 = tracer.Start(ctx, "segment.child")
				defer span2.End()

				span2.RecordError(errors.New("something bad just happened"))
				span2.SetStatus(codes.Error, "")
			}()
		}()
	}

	traces, endpoint, err := j.GetTraces(1, 30, serviceName)
	require.NoError(t, err, "must be able to get traces")

	assert.NotEmpty(t, endpoint, "must return an endpoint")
	require.Len(t, traces, 1)
	require.Len(t, traces[0].Spans, 2)
	spanMap := map[string]Span{}
	for _, s := range traces[0].Spans {
		spanMap[s.OperationName] = s
	}
	{ // outer span
		assert.Equal(t, "segment.parent", spanMap["segment.parent"].OperationName)
		assert.Equal(t, span1.SpanContext().SpanID().String(), spanMap["segment.parent"].SpanID)
		assert.Equal(t, span1.SpanContext().TraceID().String(), spanMap["segment.parent"].TraceID)

		assert.Empty(t, spanMap["segment.parent"].References)
		assert.Len(t, spanMap["segment.parent"].Tags, 2)

		{ // logs
			require.Len(t, spanMap["segment.parent"].Logs, 1)
			require.Len(t, spanMap["segment.parent"].Logs[0].Fields, 1)

			assert.Contains(t, spanMap["segment.parent"].Logs[0].Fields, KeyValue{
				Key:   "event",
				Type:  "string",
				Value: any("this is a log event"),
			})
		}
	}

	{ // inner span
		assert.Equal(t, "segment.child", spanMap["segment.child"].OperationName)
		assert.Equal(t, span2.SpanContext().SpanID().String(), spanMap["segment.child"].SpanID)
		assert.Equal(t, span2.SpanContext().TraceID().String(), spanMap["segment.child"].TraceID)

		assert.Equal(t, Reference{
			RefType: "CHILD_OF",
			TraceID: span1.SpanContext().TraceID().String(),
			SpanID:  span1.SpanContext().SpanID().String(),
		}, spanMap["segment.child"].References[0])

		{ // tags
			require.Len(t, spanMap["segment.child"].Tags, 4)

			assert.Contains(t, spanMap["segment.child"].Tags, KeyValue{
				Key:   "error",
				Type:  "bool",
				Value: any(true),
			})

			assert.Contains(t, spanMap["segment.child"].Tags, KeyValue{
				Key:   "otel.scope.name",
				Type:  "string",
				Value: any(serviceName),
			})

			assert.Contains(t, spanMap["segment.child"].Tags, KeyValue{
				Key:   "otel.status_code",
				Type:  "string",
				Value: any("ERROR"),
			})
		}

		{ // logs
			require.Len(t, spanMap["segment.child"].Logs, 1)
			require.Len(t, spanMap["segment.child"].Logs[0].Fields, 3)

			assert.Contains(t, spanMap["segment.child"].Logs[0].Fields, KeyValue{
				Key:   "event",
				Type:  "string",
				Value: any("exception"),
			})

			assert.Contains(t, spanMap["segment.child"].Logs[0].Fields, KeyValue{
				Key:   "exception.message",
				Type:  "string",
				Value: any("something bad just happened"),
			})

			assert.Contains(t, spanMap["segment.child"].Logs[0].Fields, KeyValue{
				Key:   "exception.type",
				Type:  "string",
				Value: any("*errors.fundamental"),
			})
		}
	}
}
