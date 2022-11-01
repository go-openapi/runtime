package client

import (
	"net/http"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

type config struct {
	Tracer            trace.Tracer
	Propagators       propagation.TextMapPropagator
	SpanStartOptions  []trace.SpanStartOption
	SpanNameFormatter func(string, *http.Request) string
	TracerProvider    trace.TracerProvider
}

type OpenTelemetryOption interface {
	apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(provider trace.TracerProvider) OpenTelemetryOption {
	return optionFunc(func(c *config) {
		if provider != nil {
			c.TracerProvider = provider
		}
	})
}

// WithPropagators configures specific propagators. If this
// option isn't specified, then the global TextMapPropagator is used.
func WithPropagators(ps propagation.TextMapPropagator) OpenTelemetryOption {
	return optionFunc(func(c *config) {
		if ps != nil {
			c.Propagators = ps
		}
	})
}

// WithSpanOptions configures an additional set of
// trace.SpanOptions, which are applied to each new span.
func WithSpanOptions(opts ...trace.SpanStartOption) OpenTelemetryOption {
	return optionFunc(func(c *config) {
		c.SpanStartOptions = append(c.SpanStartOptions, opts...)
	})
}

type openTelemetryTransport struct {
	transport        runtime.ClientTransport
	host             string
	spanStartOptions []trace.SpanStartOption
	propagator       propagation.TextMapPropagator
	provider         trace.TracerProvider
	tracer           trace.Tracer
	config           *config
}

// newConfig creates a new config struct and applies opts to it.
func newConfig(opts ...OpenTelemetryOption) *config {
	c := &config{
		Propagators: otel.GetTextMapPropagator(),
	}

	for _, opt := range opts {
		opt.apply(c)
	}

	// Tracer is only initialized if manually specified. Otherwise, can be passed with the tracing context.
	if c.TracerProvider != nil {
		c.Tracer = newTracer(c.TracerProvider)
	}

	return c
}

func newOpenTelemetryTransport(transport runtime.ClientTransport, host string, opts []OpenTelemetryOption) *openTelemetryTransport {
	t := &openTelemetryTransport{
		transport:  transport,
		host:       host,
		provider:   otel.GetTracerProvider(),
		propagator: otel.GetTextMapPropagator(),
	}

	c := newConfig(opts...)
	t.config = c

	return t
}

func (t *openTelemetryTransport) Submit(op *runtime.ClientOperation) (interface{}, error) {
	if op.Context == nil {
		return t.transport.Submit(op)
	}

	params := op.Params
	reader := op.Reader

	var span trace.Span
	defer func() {
		if span != nil {
			span.End()
		}
	}()

	op.Params = runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, reg strfmt.Registry) error {
		span = t.newOpenTelemetrySpan(op, req.GetHeaderParams())
		return params.WriteToRequest(req, reg)
	})

	op.Reader = runtime.ClientResponseReaderFunc(func(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
		if span != nil {
			statusCode := response.Code()
			span.SetAttributes(attribute.Int(string(semconv.HTTPStatusCodeKey), statusCode))
			span.SetStatus(semconv.SpanStatusFromHTTPStatusCodeAndSpanKind(statusCode, trace.SpanKindClient))
		}

		return reader.ReadResponse(response, consumer)
	})

	submit, err := t.transport.Submit(op)
	if err != nil && span != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return submit, err
}

func (t *openTelemetryTransport) newOpenTelemetrySpan(op *runtime.ClientOperation, header http.Header) trace.Span {
	ctx := op.Context

	tracer := t.tracer
	if tracer == nil {
		if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
			tracer = newTracer(span.TracerProvider())
		} else {
			tracer = newTracer(otel.GetTracerProvider())
		}
	}

	ctx, span := tracer.Start(ctx, operationName(op), t.spanStartOptions...)

	// TODO: Can we get the underlying request so we can wire these bits up easily?
	// span.SetAttributes(semconv.HTTPClientAttributesFromHTTPRequest()...)
	var scheme string
	if len(op.Schemes) == 1 {
		scheme = op.Schemes[0]
	}

	span.SetAttributes(
		attribute.String("net.peer.name", t.host),
		// attribute.String("net.peer.port", ""),
		attribute.String(string(semconv.HTTPRouteKey), op.PathPattern),
		attribute.String(string(semconv.HTTPMethodKey), op.Method),
		attribute.String("span.kind", trace.SpanKindClient.String()),
		attribute.String("http.scheme", scheme),
	)

	carrier := propagation.HeaderCarrier(header)
	t.propagator.Inject(ctx, carrier)

	return span
}

func newTracer(tp trace.TracerProvider) trace.Tracer {
	return tp.Tracer("go-runtime", trace.WithInstrumentationVersion("1.0.0"))
}
