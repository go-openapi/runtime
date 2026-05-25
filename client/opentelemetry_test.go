// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"net/http"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
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
	// With op.Context nil, Submit now defaults to context.Background()
	// and still produces a span (under the modern SubmitContext path).
	// Previously this case skipped tracing entirely.
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

	assertOpenTelemetrySubmit(t, operation, exporter, 1)
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

// Test_OpenTelemetryRuntime_submitContext exercises the modern
// SubmitContext entry point with a wrapped transport that satisfies
// [ContextualTransport]. The wrapper must forward the explicit ctx
// via the wrapped transport's SubmitContext (not via the legacy
// op.Context field).
func Test_OpenTelemetryRuntime_submitContext(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithSampler(tracesdk.AlwaysSample()),
		tracesdk.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)

	tracer := tp.Tracer("go-runtime")
	ctx, span := tracer.Start(context.Background(), "op")
	defer span.End()

	header := map[string][]string{}
	mock := &mockContextualRuntime{mockRuntime: mockRuntime{req: runtime.TestClientRequest{Headers: header}}}
	tr := newOpenTelemetryTransport(mock, "remote_host", nil)

	// Sentinel in op.Context: the SubmitContext path must ignore
	// it (no read for tracing, no mutation) and forward `ctx` instead.
	type sentinelKey struct{}
	staleCtx := context.WithValue(context.Background(), sentinelKey{}, "stale")
	op := testOperation(staleCtx)
	_, err := tr.SubmitContext(ctx, op)
	require.NoError(t, err)

	assert.EqualT(t, 1, mock.submitContextCalls, "wrapper must call SubmitContext on a ContextualTransport")
	assert.EqualT(t, 0, mock.submitCalls, "legacy Submit must not be called when SubmitContext is available")
	assert.Equal(t, ctx, mock.lastSubmitCtx, "ctx must be forwarded verbatim")
	assert.Equal(t, staleCtx, mock.lastOpCtx, "op.Context must not be mutated under the SubmitContext path")
	assert.Len(t, exporter.GetSpans(), 1)
}

// Test_OpenTelemetryRuntime_submitContext_legacyFallback verifies the
// fallback path when the wrapped transport implements only the legacy
// Submit method: the wrapper stamps ctx onto op.Context for the call
// and restores the prior value afterwards.
func Test_OpenTelemetryRuntime_submitContext_legacyFallback(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithSampler(tracesdk.AlwaysSample()),
		tracesdk.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)

	tracer := tp.Tracer("go-runtime")
	ctx, span := tracer.Start(context.Background(), "op")
	defer span.End()

	header := map[string][]string{}
	mock := &mockRuntime{req: runtime.TestClientRequest{Headers: header}}
	tr := newOpenTelemetryTransport(mock, "remote_host", nil)

	prevCtx, prevSpan := tracer.Start(context.Background(), "prev")
	defer prevSpan.End()
	op := testOperation(prevCtx)
	_, err := tr.SubmitContext(ctx, op)
	require.NoError(t, err)

	assert.Equal(t, prevCtx, op.Context, "op.Context must be restored after the wrapped Submit returns")
	assert.Len(t, exporter.GetSpans(), 1)
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
	assert.EqualT(t, "getCluster", span.Name)
	assert.EqualT(t, "go-openapi", span.InstrumentationScope.Name)
	assert.EqualT(t, codes.Error, span.Status.Code)
	assert.Equal(t, []attribute.KeyValue{
		attribute.String("net.peer.name", "remote_host"),
		attribute.String("http.route", "/kubernetes-clusters/{cluster_id}"),
		attribute.String("http.request.method", http.MethodGet),
		attribute.String("span.kind", trace.SpanKindClient.String()),
		attribute.String("http.scheme", schemeHTTPS),
		// NOTE: this becomes http.response.status_code with semconv v1.21
		attribute.Int("http.response.status_code", 490),
	}, span.Attributes)
}
