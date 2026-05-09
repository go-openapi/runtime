# client-middleware/opentracing

[![GoDoc][godoc-badge]][godoc-url]

OpenTracing instrumentation for the `go-openapi/runtime` client transport.

> **Compatibility module.** This module exists solely to support users who
> still depend on the legacy [OpenTracing API][opentracing-url] and have not
> yet migrated to OpenTelemetry. New code should use the **OpenTelemetry
> tracing built into [`client.Runtime`](../../client)** directly — it
> requires no extra wrapper.
>
> The OpenTracing project has been archived since 2022 in favour of
> OpenTelemetry. We expect to keep this module compiling and passing tests,
> but it will not gain new features.

It is published as a **separate Go module** so that the
`opentracing-go` dependency stays out of the main runtime's import graph —
projects on OpenTelemetry pay no cost for it.

## Install

```sh
go get github.com/go-openapi/runtime/client-middleware/opentracing
```

## Usage

`WithOpenTracing` wraps a `client.Runtime` and starts a child span for each
outgoing operation, provided the operation's `context.Context` already
carries a parent span. If no parent span is found in the context, the call
is forwarded unchanged.

```go
import (
    "github.com/go-openapi/runtime/client"
    otmw "github.com/go-openapi/runtime/client-middleware/opentracing"
)

rt := client.New("api.example.com", "/v1", []string{"https"})
transport := otmw.WithOpenTracing(rt)

// pass `transport` (a runtime.ClientTransport) to your generated client
```

Per-request tags can be added through the variadic `opentracing.StartSpanOption`
arguments:

```go
transport := otmw.WithOpenTracing(rt, opentracing.Tag{Key: "service", Value: "billing"})
```

## Migrating to OpenTelemetry

The main `client.Runtime` already emits OpenTelemetry spans; no wrapper is
needed. Users coming from this module typically:

1. Replace the `opentracing-go` `Tracer` setup with an OpenTelemetry
   `TracerProvider`.
2. Drop the `WithOpenTracing` wrapper — instrument the `client.Runtime` directly.
3. Remove the import of this module.

See the [main runtime README](../../README.md) for the OpenTelemetry
configuration entry points.

## License

[Apache-2.0](../../LICENSE).

[godoc-badge]: https://pkg.go.dev/badge/github.com/go-openapi/runtime/client-middleware/opentracing
[godoc-url]: https://pkg.go.dev/github.com/go-openapi/runtime/client-middleware/opentracing
[opentracing-url]: https://github.com/opentracing/opentracing-go
