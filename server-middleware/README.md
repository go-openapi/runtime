# server-middleware

[![GoDoc][godoc-badge]][godoc-url]

Standalone, dependency-free server-side middleware utilities for OpenAPI applications.

This module is part of the [`go-openapi/runtime`][runtime-url] toolkit, but is
maintained as a **separate Go module** so it can be used by any `net/http`
application — including ones that have no OpenAPI spec at all. It carries no
transitive dependency on `go-openapi/spec`, `go-openapi/loads`, or
`go-openapi/validate`; only the standard library and (for tests)
`go-openapi/testify/v2`.

## Packages

| Package | Purpose |
|---------|---------|
| [`mediatype`](./mediatype) | Typed RFC 7231 / RFC 2045 media-type values (`MediaType`, `Set`) and asymmetric matching used by both server-side validation and `Accept`-header negotiation. |
| [`negotiate`](./negotiate) | Server-side HTTP content negotiation: select the response `Content-Type` from `Accept`, and the response `Content-Encoding` from `Accept-Encoding`. Honours MIME parameters by default; opt out with `WithIgnoreParameters`. |
| [`negotiate/header`](./negotiate/header) | Low-level RFC-7231 header parsing primitives reused by `negotiate`. Exported for callers that need raw `Accept`/`Accept-Encoding` parsing without the typed media-type layer. |
| [`docui`](./docui) | Stdlib-only HTTP middlewares that serve OpenAPI documentation UIs (Swagger UI, ReDoc, RapiDoc) and the spec document itself. Mountable on any `net/http` mux. |

## Install

```sh
go get github.com/go-openapi/runtime/server-middleware
```

## Quick start — content negotiation

```go
import "github.com/go-openapi/runtime/server-middleware/negotiate"

offers := []string{"application/json", "application/xml"}
chosen := negotiate.ContentType(r.Header, offers, "application/json")
w.Header().Set("Content-Type", chosen)
```

## Quick start — serving Swagger UI

```go
import "github.com/go-openapi/runtime/server-middleware/docui"

handler := docui.SwaggerUI(docui.WithSpecURL("/swagger.json"), docui.WithBasePath("/docs"))
http.Handle("/docs/", handler)
```

## Further reading

- [media-type selection tutorial](https://go-openapi.github.io/runtime/tutorials/media-types/) — the full server-side
  selection algorithm and its asymmetric matching rules.
- Full API reference on [pkg.go.dev][godoc-url].

## License

[Apache-2.0](../LICENSE).

[runtime-url]: https://github.com/go-openapi/runtime
[godoc-badge]: https://pkg.go.dev/badge/github.com/go-openapi/runtime/server-middleware
[godoc-url]: https://pkg.go.dev/github.com/go-openapi/runtime/server-middleware
