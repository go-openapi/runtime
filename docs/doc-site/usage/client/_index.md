---
title: Client
weight: 20
description: |
  HTTP client transport — TLS, auth, OpenTelemetry tracing and
  request submission for go-swagger-generated and untyped API clients.
---

The `client` package provides
[`client.Runtime`](https://pkg.go.dev/github.com/go-openapi/runtime/client#Runtime),
the configurable HTTP transport that go-swagger-generated clients use
under the hood. You can also drive it directly for untyped API calls.

A minimal client looks like this:

{{< code file="client/intro/main.go" lang="go" region="minimalClient" >}}

What the four pages below cover:

| Page                                        | About                                                                       |
|---------------------------------------------|-----------------------------------------------------------------------------|
| [Transport](./transport/)                   | `Runtime` configuration: TLS, timeouts, proxy, keepalive, debug logging     |
| [Authentication](./auth/)                   | `ClientAuthInfoWriter` and the built-in helpers (Basic, API key, Bearer)    |
| [Tracing](./tracing/)                       | OpenTelemetry support and the legacy OpenTracing compat module              |
| [Building & submitting requests](./requests/) | `ClientOperation`, `Submit` vs `SubmitContext`, the v0.30 context-only pivot |

{{< children type="card" description="true" >}}
