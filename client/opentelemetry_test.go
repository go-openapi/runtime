package client

import (
	"context"
	"testing"

	"github.com/go-openapi/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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
	ctx, _ := tracer.Start(context.Background(), "op")

	testOpenTelemetrySubmit(t, testOperation(ctx), exporter, 1)
}

// func Test_OpenTelemetryRuntime_submit_nilAuthInfo(t *testing.T) {
// 	t.Parallel()
// 	tracer := mocktracer.New()
// 	_, ctx := opentracing.StartSpanFromContextWithTracer(context.Background(), tracer, "op")
// 	operation := testOperation(ctx)
// 	operation.AuthInfo = nil
// 	testOpenTelemetrySubmit(t, operation, tracer, 1)
// }

// func Test_OpenTelemetryRuntime_submit_nilContext(t *testing.T) {
// 	t.Parallel()
// 	tracer := mocktracer.New()
// 	_, ctx := opentracing.StartSpanFromContextWithTracer(context.Background(), tracer, "op")
// 	operation := testOperation(ctx)
// 	operation.Context = nil
// 	testOpenTelemetrySubmit(t, operation, tracer, 0) // just don't panic
// }

// func Test_injectOpenTelemetrySpanContext(t *testing.T) {
// 	t.Parallel()
// 	tracer := mocktracer.New()
// 	_, ctx := opentracing.StartSpanFromContextWithTracer(context.Background(), tracer, "op")
// 	header := map[string][]string{}
// 	createOpenTelemetryClientSpan(testOperation(ctx), header, "", nil)

// 	// values are random - just check that something was injected
// 	assert.Len(t, header, 3)
// }

func testOpenTelemetrySubmit(t *testing.T, operation *runtime.ClientOperation, exporter *tracetest.InMemoryExporter, expectedSpanCount int) {
	header := map[string][]string{}
	r := newOpenTelemetryTransport(&mockRuntime{runtime.TestClientRequest{Headers: header}},
		"remote_host",
		[]trace.SpanStartOption{})

	// // opentracing.Tag{
	// 	Key:   string(ext.PeerService),
	// 	Value: "service",
	// }

	_, err := r.Submit(operation)
	require.NoError(t, err)

	spans := exporter.GetSpans()
	assert.Len(t, spans, expectedSpanCount)

	if expectedSpanCount == 1 {
		span := spans[0]
		assert.Equal(t, "getCluster", span.Name)
		assert.Equal(t, []attribute.KeyValue{
			// "component":        "go-openapi",
			attribute.String("http.path", "/kubernetes-clusters/{cluster_id}"),
			attribute.String("http.method", "GET"),
			attribute.String("span.kind", trace.SpanKindClient.String()),
			// // "http.status_code": uint16(490),
			// "peer.hostname": "remote_host",
			// "peer.service":  "service",
			// "error":         true,
		}, span.Attributes)
	}
}
