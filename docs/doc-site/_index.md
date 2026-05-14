---
title: "go-openapi runtime"
type: home
description: HTTP runtime for OpenAPI clients and servers in Go.
weight: 1
---

`github.com/go-openapi/runtime` is a runtime library used to work with OpenAPI.

At this moment, it only supports OpenAPI v2 (aka Swagger).

It is used by clients and servers generated with [go-swagger][go-swagger].
or directly by applications that build untyped OpenAPI / Swagger clients or servers.

It ships:

* a configurable HTTP **client transport** (`client.Runtime`) — TLS, proxy,
  timeouts, OpenTelemetry tracing, pluggable authentication
* a **server middleware pipeline** that turns an analyzed OpenAPI spec into a
  working `http.Handler` — routing, security, parameter binding, validation
  and operation execution
* a **dependency-free server-middleware module** with media-type processing, content
  negotiation and doc-UI helpers, usable from any plain `net/http` server

### Status

{{% button href="https://github.com/go-openapi/runtime/fork" hint="fork me on github" style=primary icon=code-fork %}}Fork me{{% /button %}}
Stable API. Actively maintained.

<!-- See our [ROADMAP](./project/maintainers/ROADMAP.md). -->

### Getting started

```cmd
go get github.com/go-openapi/runtime
```

Using only the dependency-free middleware (media types, negotiation, doc UIs):

```cmd
go get github.com/go-openapi/runtime/server-middleware
```

### Where to go next

{{< cards >}}
{{% card title="Features" %}}
Features supported by our client and server, with normative references.
→ [usage/features](./usage/features/)
{{% /card %}}
{{% card title="Core" %}}
The five interfaces (`Consumer`, `Producer`, `Authenticator`, `Authorizer`,
`OperationHandler`) every other piece is built on, plus content-type and
validation plumbing.

→ [usage/core](./usage/core/)
{{% /card %}}

{{% card title="Client" %}}
Configuring `client.Runtime` for TLS, auth, OpenTelemetry tracing and
context-aware request submission.

→ [usage/client](./usage/client/)
{{% /card %}}

{{% card title="Server" %}}
The Router → Binder → Validator → Security → OperationExecutor → Responder
pipeline that turns a spec into a handler.

→ [usage/server](./usage/server/)
{{% /card %}}

{{% card title="Standalone" %}}
Use the media-type, content-negotiation and doc-UI helpers from any plain
`net/http` server, with no transitive OpenAPI dependencies.

→ [usage/standalone](./usage/standalone/)
{{% /card %}}
{{< /cards >}}

Looking for runnable code? See [examples](./usage/examples/).

## Licensing

`SPDX-FileCopyrightText: Copyright 2025 go-swagger maintainers`

This library ships under the [Apache-2.0 license](./project/LICENSE.md).

## Contributing

Issues and pull requests welcome.

See the shared [go-openapi contributing guidelines][contributing-doc-site] and
the per-repo notes in [project/](./project/).

---

{{< children type="card" description="true" >}}

[go-swagger]: https://github.com/go-swagger/go-swagger
[contributing-doc-site]: https://go-openapi.github.io/doc-site/contributing/contributing/index.html
[maintainers-doc-site]: https://go-openapi.github.io/doc-site/maintainers/index.html
