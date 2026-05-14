---
title: Core
weight: 10
description: |
  The interfaces, content-type plumbing and validation hooks the
  client and server pieces are built on. Start here for a mental model.
---

The root `github.com/go-openapi/runtime` package defines a small set of
interfaces shared by every other piece of the runtime.

Everything else — client transport, server middleware pipeline — is built on top of these.

## Concepts

* `Producer` and `Consumer` are the _codecs_ that map data structures to and from JSON, YAML, XML, byte streams, etc.
* "Parameters bindings" is the machinery to serialize / deserialize OpenAPI request parameters as go types
* "Content negotiation" refers to the handling of the `Content-Type` header to agree on a serialization and encoding format.
* "Operation" is the OpenAPI term for "handler", i.e. a unitary service invoked by a request

## Module map

`runtime` ships as **three Go modules**, each with its own `go.mod` and dependencies.

| Module                                                                          | Purpose                                                                                                                                            |
|---------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------|
| `github.com/go-openapi/runtime`                                                 | Core interfaces, content-type codecs, client transport, full server middleware pipeline. Pulls in `analysis`, `loads`, `spec`, `strfmt`, `validate`. |
| `github.com/go-openapi/runtime/server-middleware`                               | Standalone, dependency-free server middleware: media types, content negotiation, doc UIs. Usable from any plain `net/http` server.                 |
| `github.com/go-openapi/runtime/client-middleware/opentracing`                   | Optional OpenTracing transport middleware (compatibility add-on — new code should use the OpenTelemetry support built into `client.Runtime`).      |

> `server-middleware` lets you reuse the negotiation
> and doc-UI primitives without inheriting the OpenAPI spec/loads/validate
> dependency tree.

## Where the pieces fit

{{< mermaid align="center" zoom="true" >}}
flowchart TD
    cli["Application client code<br/>(models, …)"]
    app["Application server code<br/>(handlers, models, …)"]
    client[["client<br/>(transport)"]]
    mw[["middleware<br/>(pipeline)"]]
    sm[["server-middleware<br/>(standalone — stdlib only)"]]
    core{{"runtime<br/>core interfaces<br/>Consumer · Producer<br/>Authenticator · Authorizer<br/>OperationHandler · Validatable"}}

    cli -- import --> client
    app -- import --> mw
    app --> sm
    client --> core
    mw --> core
    mw -.-> sm
{{< /mermaid >}}


> `middleware` reuses the `server-middleware` primitives (the dotted
> arrow): negotiation, media-type matching and the doc-UI handlers all
> live in `server-middleware`.

> **Backward-compatibility note**
> 
> The legacy entry points pre-existing in package `middleware` before v0.30.0 (`NegotiateContentType`, `SwaggerUI`, …)
> are still available as a shim (`middleware/seam.go`) that now forwards to
> the new module — see [server / deprecated shims](../../server/deprecated-shims/).

Read on for what each interface does, the built-in content-type codecs and
the validation hooks.

{{< children type="card" description="true" >}}
