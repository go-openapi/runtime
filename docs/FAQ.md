# Frequently Asked Questions

Answers to common questions collected from [GitHub issues](https://github.com/go-openapi/runtime/issues).

---

## Client

### How do I disable TLS certificate verification?

Use `TLSClientOptions` with `InsecureSkipVerify`:

```go
import "github.com/go-openapi/runtime/client"

httpClient, err := client.TLSClient(client.TLSClientOptions{
    InsecureSkipVerify: true,
})
```

Then pass the resulting `*http.Client` to your transport.

> [#196](https://github.com/go-openapi/runtime/issues/196)

### Why is `request.ContentLength` zero when I send a body?

A streaming body (e.g. from `bytes.NewReader`) is sent with chunked transfer encoding.
The runtime cannot know the content length of an arbitrary stream unless you explicitly
set it on the request. If you need `ContentLength` populated, set it yourself before
submitting.

> [#253](https://github.com/go-openapi/runtime/issues/253)

### How do I read the error response body from an `APIError`?

The client's `Submit()` closes the response body after reading. To access error details,
define your error responses (including a `default` response) in the Swagger spec with a
schema. The generated client will then deserialize the error body into a typed struct
that you can access via type assertion:

```go
if apiErr, ok := err.(*mypackage.GetThingDefault); ok {
    // apiErr.Payload contains the deserialized error body
}
```

Without a response schema in the spec, the body is discarded and only the status code
is available in the `runtime.APIError`.

> [#89](https://github.com/go-openapi/runtime/issues/89), [#121](https://github.com/go-openapi/runtime/issues/121)

### How do I register custom MIME types (e.g. `application/problem+json`)?

The default client runtime ships with a fixed set of consumers/producers. Register
custom ones on the transport:

```go
rt := client.New(host, basePath, schemes)
rt.Consumers["application/problem+json"] = runtime.JSONConsumer()
rt.Producers["application/problem+json"] = runtime.JSONProducer()
```

The same approach works for any non-standard MIME type such as `application/pdf`
(use `runtime.ByteStreamConsumer()`), `application/hal+json`, or
`application/vnd.error+json` (use `runtime.JSONConsumer()`).

> [#31](https://github.com/go-openapi/runtime/issues/31), [#252](https://github.com/go-openapi/runtime/issues/252), [#329](https://github.com/go-openapi/runtime/issues/329)

---

## Middleware

### How do I access the authenticated Principal from an `OperationHandler`?

Use the context helpers from the `middleware` package:

```go
func myHandler(r *http.Request, params MyParams) middleware.Responder {
    principal := middleware.SecurityPrincipalFrom(r)
    route      := middleware.MatchedRouteFrom(r)
    scopes     := middleware.SecurityScopesFrom(r)
    // ...
}
```

These extract values that the middleware pipeline stored in the request context
during authentication and routing.

> [#203](https://github.com/go-openapi/runtime/issues/203)

### Can I run authentication on requests that don't match a route?

No. Authentication is determined dynamically per route from the OpenAPI spec
(each operation declares its own security requirements). The middleware pipeline
authenticates *after* routing, so unmatched requests are never authenticated.

> [#201](https://github.com/go-openapi/runtime/issues/201)

### How do I share context values across middlewares when using an external router?

The go-openapi router creates a new request context during route resolution.
Context values set after routing (e.g. during auth) are not visible to middlewares
that run before the router in the chain.

The recommended pattern is to use a pointer-based shared struct:

```go
type sharedCtx struct {
    Principal any
    // add fields as needed
}

// In your outermost middleware, before the router:
sc := &sharedCtx{}
ctx := context.WithValue(r.Context(), sharedCtxKey, sc)
next.ServeHTTP(w, r.WithContext(ctx))
// After ServeHTTP returns, sc is populated by inner middlewares.

// In an inner middleware or auth handler:
sc := r.Context().Value(sharedCtxKey).(*sharedCtx)
sc.Principal = principal // visible to the outer middleware
```

Because the struct is shared by pointer, mutations are visible regardless of
which request copy carries the context.

> [#375](https://github.com/go-openapi/runtime/issues/375)

### Can I use this library to validate requests/responses without code generation?

Yes. Use the routing and validation middleware from the `middleware` package with
an untyped API. Load your spec with `loads.Spec()`, then wire up
`middleware.NewRouter()` to get request validation against the spec without
needing go-swagger generated code. See the `middleware/untyped` package for
examples.

> [#44](https://github.com/go-openapi/runtime/issues/44)

### How do I configure Swagger UI to show multiple specs?

`SwaggerUIOpts` supports the `urls` parameter for listing multiple spec files in
the Swagger UI explore bar. Configure it instead of the single `url` parameter.

> [#316](https://github.com/go-openapi/runtime/issues/316)

---

## Documentation

### Where can I find middleware documentation?

- [GoDoc](https://pkg.go.dev/github.com/go-openapi/runtime/middleware) — API reference
- [go-swagger middleware guide](https://goswagger.io/use/middleware.html) — usage patterns
- [go-swagger FAQ](https://goswagger.io/faq/) — common questions

> [#82](https://github.com/go-openapi/runtime/issues/82)
