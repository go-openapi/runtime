---
title: Tracing
weight: 30
description: |
  Built-in OpenTelemetry support on client.Runtime, plus a note on the
  legacy OpenTracing compatibility module.
---

`client.Runtime` ships first-class OpenTelemetry support. There are
no extra modules to import beyond the runtime itself
(it already depends on `go.opentelemetry.io/otel`).

## Wire it up — `WithOpenTelemetry`

See [`client.Runtime.WithOpenTelemetry`](https://pkg.go.dev/github.com/go-openapi/runtime/client#Runtime.WithOpenTelemetry)
for the authoritative signature:

    func (r *Runtime) WithOpenTelemetry(opts ...OpenTelemetryOpt) runtime.ClientTransport

Returns a `runtime.ClientTransport` that delegates to the underlying
runtime and creates a client span for every request. Use it as the
transport you hand to a generated client:

{{< code file="client/tracing/main.go" lang="go" region="wireOpenTelemetry" >}}

For untyped use you call `traced.Submit(op)` directly.

### A span only appears when one is already active

If the operation's context does not contain an active span, the
transport does **not** start a root span. This is intentional —
telemetry boundaries belong to the application, not to the transport
library. Wrap your call site in a span and the client span attaches
beneath it.

## Options — `OpenTelemetryOpt`

| Option                         | What it sets                                                                                       | Default                                          |
|--------------------------------|----------------------------------------------------------------------------------------------------|--------------------------------------------------|
| `WithTracerProvider(provider)` | The `trace.TracerProvider` to acquire a tracer from.                                               | the global provider (`otel.GetTracerProvider`)   |
| `WithPropagators(ps)`          | The `propagation.TextMapPropagator` used to inject context into outbound headers.                  | the global propagator (`otel.GetTextMapPropagator`) |
| `WithSpanOptions(opts…)`       | Extra `trace.SpanStartOption`s applied to every new span (kind, attributes, etc.).                 | none                                             |
| `WithSpanNameFormatter(fn)`    | Function that derives the span name from the `*runtime.ClientOperation`.                          | `op.ID` if non-empty, otherwise `"{method}_{pathPattern}"` |

Example with a custom name and global tags:

{{< code file="client/tracing/main.go" lang="go" region="customSpanFormatter" >}}

## Legacy OpenTracing

`Runtime.WithOpenTracing` exists but is **deprecated**. It silently
returns an OpenTelemetry transport, ignoring opts that are not
`OpenTelemetryOpt`. The OpenTracing project is archived — new code
should call `WithOpenTelemetry`.

If you still need OpenTracing semantics (for example because your
collector is OpenTracing-only), import the compatibility add-on:

```sh
go get github.com/go-openapi/runtime/client-middleware/opentracing
```

```go
import (
    "github.com/go-openapi/runtime/client-middleware/opentracing"
    ottrace "github.com/opentracing/opentracing-go"
)

traced := opentracing.WithOpenTracing(rt, ottrace.GlobalTracer())
```

The compat module lives in its own Go module so the rest of the
runtime no longer pulls the OpenTracing dependency.
