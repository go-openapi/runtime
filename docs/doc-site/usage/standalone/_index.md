---
title: Standalone middleware
weight: 40
description: |
  The dependency-free server-middleware module — media types, content
  negotiation and doc-UI handlers, usable from any net/http server.
---

`github.com/go-openapi/runtime/server-middleware` is a separate Go module
that ships the negotiation, media-type and doc-UI primitives without
inheriting the OpenAPI spec / loads / validate dependency tree. Drop it
into any vanilla `net/http` application.

## Install

```sh
go get github.com/go-openapi/runtime/server-middleware
```

## What's in it

| Package                                                                                                | Use it for                                                                                                                                                              |
|--------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [`mediatype`](https://pkg.go.dev/github.com/go-openapi/runtime/server-middleware/mediatype)            | Parsed RFC 7231 media-type values, `Set` lists and asymmetric matching — the building block both `negotiate` and the runtime's own server pipeline use.                 |
| [`negotiate`](https://pkg.go.dev/github.com/go-openapi/runtime/server-middleware/negotiate)            | Server-side selection from `Accept` via `ContentType`. Honours MIME parameters by default; opt out with `WithIgnoreParameters`.                                         |
| [`negotiate/header`](https://pkg.go.dev/github.com/go-openapi/runtime/server-middleware/negotiate/header) | Low-level RFC 7231 header parsing primitives reused by `negotiate`. Use it directly if you need raw `Accept`/`Accept-Encoding` parsing without the typed media-type layer. |
| [`docui`](https://pkg.go.dev/github.com/go-openapi/runtime/server-middleware/docui)                    | Stdlib-only handlers that serve Swagger UI, RapiDoc or Redoc, plus the spec document itself. Mountable on any `net/http` mux.                                           |

The module has zero transitive dependencies on `go-openapi/spec`,
`go-openapi/loads`, `go-openapi/validate`, or even on the rest of
`go-openapi/runtime`. Standard library only.

{{< children type="card" description="true" >}}
