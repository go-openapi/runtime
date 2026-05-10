---
title: Building & submitting requests
weight: 40
description: |
  ClientOperation, BuildHTTP and SubmitContext — and the recent pivot
  to context-only request building.
---

`Runtime` exposes a small set of entry points for turning a
`runtime.ClientOperation` into a sent request and a typed result. The
public surface has been pivoting from "use the cached context on the
operation/runtime" to "pass the context explicitly". This page covers
both shapes and explains which to use when.

## The descriptor — `ClientOperation`

Authoritative definitions live in the
[`runtime`](https://pkg.go.dev/github.com/go-openapi/runtime#ClientOperation)
package:

```go
package runtime

type ClientOperation struct {
    ID                 string
    Method             string
    PathPattern        string
    ProducesMediaTypes []string
    ConsumesMediaTypes []string
    Schemes            []string
    AuthInfo           ClientAuthInfoWriter
    Params             ClientRequestWriter
    Reader             ClientResponseReader
    Context            context.Context // legacy — see below
    Client             *http.Client    // optional per-call override
}

type ClientTransport interface {
    Submit(*ClientOperation) (any, error)
}
```

Generated clients build one of these per operation method and call
`Submit` (or, increasingly, `SubmitContext`). For untyped use you
populate the fields by hand.

## Entry points

The runtime offers four methods, paired by purpose:

| Purpose                                  | Legacy (cached ctx)                | Context-aware (preferred)                          |
|------------------------------------------|------------------------------------|----------------------------------------------------|
| Send the request, return the typed result | `Runtime.Submit(op)`               | `Runtime.SubmitContext(ctx, op)`                   |
| Build the `*http.Request` only           | `Runtime.CreateHttpRequest(op)` ⚠  | `Runtime.CreateHTTPRequestContext(ctx, op)`        |

⚠ `CreateHttpRequest` is **deprecated**. It does not return the
context's cancel function, so any per-request timeout set via
`Params.SetTimeout` is silently leaked. Use `CreateHTTPRequestContext`
instead.

### `Submit` vs `SubmitContext`

`Submit` consults its context in this order:

1. `op.Context` if non-nil
2. otherwise `rt.Context`
3. otherwise `context.Background()`

`SubmitContext(ctx, op)` ignores those cached values entirely and uses
`ctx` as the parent context. This is the only way to pass a
caller-controlled context that can be cancelled, deadlined or
trace-instrumented from the call site.

{{< code file="client/requests/main.go" lang="go" region="submitVariants" >}}

The per-request timeout set via `Params.SetTimeout(d)` (i.e.
`runtime.ClientRequestWriter.SetTimeout`) is honoured by **both**
forms — it is applied when the request context is derived inside
`BuildHTTPContext`, on top of whatever deadline `ctx` already carries.

### Build-only — `CreateHTTPRequestContext`

When you need the prepared `*http.Request` but want to drive
`http.Client.Do` yourself (for retries, custom logging, response-body
inspection), use:

{{< code file="client/requests/main.go" lang="go" region="createHTTPRequestContext" >}}

`cancel` releases the per-request timeout timer and any other
resources held by the derived context. **Calling it before the
response body is fully drained will cancel the in-flight request** —
defer it to the end of the read.

On error the returned cancel is a no-op, so deferring it
unconditionally is safe.

## What happens during a `SubmitContext` call

{{< mermaid align="center" zoom="true" >}}
flowchart TD
    in(((SubmitContext ctx, op)))
    prep["prepareRequest<br/>resolve scheme + media type<br/>pick AuthInfoWriter (op.AuthInfo or rt.DefaultAuthentication)"]
    build["BuildHTTPContext<br/>WriteToRequest → ctx with timeout<br/>buffered or streaming body<br/>AuthenticateRequest"]
    do["http.Client.Do"]
    decode["resolveConsumer · ReadResponse<br/>decode into typed result"]
    out(((result, err)))
    cancel["cancel()<br/>(deferred)"]

    in --> prep --> build --> do --> decode --> out
    build -.-> cancel
{{< /mermaid >}}

`BuildHTTPContext` chooses one of two assembly paths:

- **buffered body** — for URL-encoded forms, producer output, or no
  body. The body is materialised in memory before `AuthenticateRequest`
  runs, so writers like HMAC signers see the final bytes.
- **streaming body** — for multipart uploads or stream payloads
  (`io.Reader` body). The body flows through an `io.Pipe`. Auth
  writers receive a body-copy closure so signers can still see the
  bytes — at the cost of one extra read.

### Multipart uploads honour context cancellation

A long-standing rough edge — the multipart upload goroutine ignoring
the request context — was fixed in `feat(client): honor context
cancellation in multipart upload goroutine`. Cancelling the context
mid-upload now stops the writer goroutine cleanly instead of leaking
it for the lifetime of the connection.

## Migration from the legacy form

If your codebase calls `Submit` and stashes contexts on `op.Context`
or `rt.Context`, the change is usually mechanical:

{{< code file="client/requests/main.go" lang="go" region="migrationForm" >}}

`op.Context` and `rt.Context` are still read by `Submit` for
compatibility with existing callers and generated code that has not
yet been regenerated; `SubmitContext` ignores both. New code (and
freshly regenerated clients) should pass the context explicitly.

For `CreateHttpRequest` callers the move is more important — the
deprecated form leaks the per-request timer when `Params.SetTimeout`
is non-zero. Switch to `CreateHTTPRequestContext` and remember to
defer the returned `cancel`.
