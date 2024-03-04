package client

import (
	"context"
	"net/http"
	"testing"

	"github.com/go-openapi/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func Test_OpenTelemetryRuntime_submit(t *testing.T) {
	t.Parallel()

	exporter := tracetest.NewInMemoryExporter()

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithSampler(tracesdk.AlwaysSample()),
		tracesdk.WithSyncer(exporter),
	)

	otel.SetTracerProvider(tp)

	tracer := tp.Tracer("go-runtime")
	ctx, span := tracer.Start(context.Background(), "op")
	defer span.End()

	assertOpenTelemetrySubmit(t, testOperation(ctx), exporter, 1)
}

func Test_OpenTelemetryRuntime_submit_nilAuthInfo(t *testing.T) {
	t.Parallel()

	exporter := tracetest.NewInMemoryExporter()

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithSampler(tracesdk.AlwaysSample()),
		tracesdk.WithSyncer(exporter),
	)

	otel.SetTracerProvider(tp)

	tracer := tp.Tracer("go-runtime")
	ctx, span := tracer.Start(context.Background(), "op")
	defer span.End()

	operation := testOperation(ctx)
	operation.AuthInfo = nil
	assertOpenTelemetrySubmit(t, operation, exporter, 1)
}

func Test_OpenTelemetryRuntime_submit_nilContext(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithSampler(tracesdk.AlwaysSample()),
		tracesdk.WithSyncer(exporter),
	)

	otel.SetTracerProvider(tp)

	tracer := tp.Tracer("go-runtime")
	ctx, span := tracer.Start(context.Background(), "op")
	defer span.End()
	operation := testOperation(ctx)
	operation.Context = nil

	assertOpenTelemetrySubmit(t, operation, exporter, 0) // just don't panic
}

func Test_injectOpenTelemetrySpanContext(t *testing.T) {
	t.Parallel()

	exporter := tracetest.NewInMemoryExporter()

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithSampler(tracesdk.AlwaysSample()),
		tracesdk.WithSyncer(exporter),
	)

	otel.SetTracerProvider(tp)

	tracer := tp.Tracer("go-runtime")
	ctx, span := tracer.Start(context.Background(), "op")
	defer span.End()
	operation := testOperation(ctx)

	header := map[string][]string{}
	tr := newOpenTelemetryTransport(&mockRuntime{runtime.TestClientRequest{Headers: header}}, "", nil)
	tr.config.Propagator = propagation.TraceContext{}
	_, err := tr.Submit(operation)
	require.NoError(t, err)

	assert.Len(t, header, 1)
}

func assertOpenTelemetrySubmit(t *testing.T, operation *runtime.ClientOperation, exporter *tracetest.InMemoryExporter, expectedSpanCount int) {
	header := map[string][]string{}
	tr := newOpenTelemetryTransport(&mockRuntime{runtime.TestClientRequest{Headers: header}}, "remote_host", nil)

	_, err := tr.Submit(operation)
	require.NoError(t, err)

	spans := exporter.GetSpans()
	assert.Len(t, spans, expectedSpanCount)

	if expectedSpanCount != 1 {
		return
	}

	span := spans[0]
	assert.Equal(t, "getCluster", span.Name)
	assert.Equal(t, "go-openapi", span.InstrumentationLibrary.Name)
	assert.Equal(t, codes.Error, span.Status.Code)
	assert.Equal(t, []attribute.KeyValue{
		attribute.String("net.peer.name", "remote_host"),
		attribute.String("http.route", "/kubernetes-clusters/{cluster_id}"),
		attribute.String("http.method", http.MethodGet),
		attribute.String("span.kind", trace.SpanKindClient.String()),
		attribute.String("http.scheme", schemeHTTPS),
		// NOTE: this becomes http.response.status_code with semconv v1.21
		attribute.Int("http.status_code", 490),
	}, span.Attributes)
}
