---
title: Request pipeline
weight: 10
description: |
  How an inbound HTTP request flows through middleware.Context — from
  routing to security to binding/validation to operation execution
  and response writing.
---

The [`middleware`](https://pkg.go.dev/github.com/go-openapi/runtime/middleware)
package wires an analyzed OpenAPI spec into a working `http.Handler`.
Every request goes through the same conventional sequence of stages —
covered briefly on [core / interfaces](../../core/interfaces/), and
expanded here with the actual call sites.

## The full picture

{{< mermaid align="center" zoom="true" >}}
flowchart TD
    req(((HTTP request)))
    router["Router · NewRouter / Context.RouteInfo<br/>match path/method against the analyzed spec<br/>404 / 405 if no route"]
    sec["Security · Context.Authorize<br/>RouteAuthenticators.Authenticate<br/>then optional Authorizer<br/>401 / 403 on failure"]
    bvr["BindValidRequest"]
    neg["ContentType / Accept negotiation<br/>pick Consumer + target Producer<br/>400 / 415 / 406 on failure"]
    bind["Binder<br/>path / query / header / body params<br/>— uses Consumer —"]
    val["Validator<br/>spec rules + Validatable<br/>422 with CompositeValidationError on failure"]
    op["OperationHandler.Handle<br/>your business logic"]
    resp["Responder · Context.Respond<br/>— uses Producer —"]
    out(((HTTP response)))

    req --> router --> sec --> bvr
    bvr --> neg --> bind --> val
    val --> op --> resp --> out
{{< /mermaid >}}

The middle three stages — negotiation, binding, validation — all live
inside the single call `Context.BindValidRequest`. Splitting them out
in the diagram makes the failure modes (400, 415, 406, 422) easier to
trace.

The diagram shows the *typical* sequence — what the runtime's
default untyped wiring does and what go-swagger's generated typed
handlers do. The actual ordering and composition is an
implementation detail of the [RoutableAPI](#the-routableapi-seam)
plugged into the `middleware.Context`; a custom one can compose the
per-route handler differently.

## The `RoutableAPI` seam

The `middleware` package handles routing, negotiation, validation
and the high-level lifecycle helpers (`RouteInfo`, `Authorize`,
`BindValidRequest`, `Respond`). Everything that has to know about
*your* API — the per-operation handler, the registered codecs, the
auth schemes — sits behind a single interface:

```go
package middleware

type RoutableAPI interface {
    HandlerFor(method, path string) (http.Handler, bool)
    ServeErrorFor(path string) func(http.ResponseWriter, *http.Request, error)
    ConsumersFor(mediaTypes []string) map[string]runtime.Consumer
    ProducersFor(mediaTypes []string) map[string]runtime.Producer
    AuthenticatorsFor(schemes map[string]spec.SecurityScheme) map[string]runtime.Authenticator
    Authorizer() runtime.Authorizer
    Formats() strfmt.Registry
    DefaultProduces() string
    DefaultConsumes() string
}
```

| Method                     | The runtime asks for…                                                      |
|----------------------------|---------------------------------------------------------------------------|
| `HandlerFor`               | the `http.Handler` that runs the per-operation pipeline for this route    |
| `ServeErrorFor`            | the error-rendering function for a given path (defaults to the API's)     |
| `ConsumersFor`             | a `mediaType → Consumer` map for the given list (route's `consumes`)      |
| `ProducersFor`             | a `mediaType → Producer` map for the given list (route's `produces`)      |
| `AuthenticatorsFor`        | a `scheme name → Authenticator` map for the security schemes in scope     |
| `Authorizer`               | the optional `Authorizer` to gate the principal post-authentication       |
| `Formats`                  | the `strfmt.Registry` used by validation                                   |
| `DefaultProduces` / `DefaultConsumes` | the API-level defaults to fall back to when the route is unspecified |

The router calls `HandlerFor(method, path)` once per matched route
and serves whatever it gets back. **What that handler does is
entirely up to the implementation** — the `RoutableAPI` decides how
the bind/validate/security/operation/respond steps are composed.

### Constructors that take a custom `RoutableAPI`

{{< code file="server/pipeline/main.go" lang="go" region="contextConstructors" >}}

Use `NewRoutableContext` when you have your own implementation
(typically the one go-swagger generates for typed APIs, but any
type satisfying the interface works). Reach for
`NewRoutableContextWithAnalyzedSpec` if you have already produced an
`*analysis.Spec` and want to avoid the second analysis pass.

### Two implementations the runtime sees in practice

The runtime ships **one** `RoutableAPI` implementation —
`routableUntypedAPI`, internal to the `middleware` package. It wraps
[`untyped.API`](https://pkg.go.dev/github.com/go-openapi/runtime/middleware/untyped#API)
and is what `middleware.Serve` / `ServeWithBuilder` builds for you.

go-swagger generates a **second** implementation per spec — the
`*operations.MyAPI` type implements every method on `RoutableAPI`
directly, with `HandlerFor` returning the per-operation `ServeHTTP`
shown below.

The next section walks both.

## Two assembly paths

The two `RoutableAPI` implementations introduced above produce
equivalent pipelines, but differ in *where* the per-route handler is
assembled — the untyped one builds it in the runtime via a closure;
the typed one is generated source you can read directly.

### Untyped — `middleware.Serve` / `ServeWithBuilder`

{{< code file="server/pipeline/main.go" lang="go" region="untypedServer" >}}

Internally `middleware.newRoutableUntypedAPI` builds one
`http.Handler` per route. The bind/validate/handle/respond logic
lives in a single closure; if the route declares any security
requirement, that closure is wrapped with `newSecureAPI` so security
runs first:

```go
// excerpt from middleware/context.go
var handler http.Handler = http.HandlerFunc(func(w, r) {
    bound, r, validation = context.BindAndValidate(r, route)
    if validation != nil { context.Respond(...); return }
    result, err := oh.Handle(bound)
    // …
})
if len(schemes) > 0 {
    handler = newSecureAPI(context, handler)   // ← wraps with Authorize
}
```

### Typed — generated `ServeHTTP` per operation

go-swagger generates a small handler per operation that calls the
same primitives in the same order, but spelt out explicitly:

```go
// excerpt from a go-swagger generated *operation.ServeHTTP
func (o *GetOrder) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
    route, rCtx, _ := o.Context.RouteInfo(r)
    if rCtx != nil { *r = *rCtx }

    var Params = NewGetOrderParams()
    uprinc, aCtx, err := o.Context.Authorize(r, route)        // ← security
    if err != nil { o.Context.Respond(rw, r, route.Produces, route, err); return }
    // …

    if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // ← bind+validate
        o.Context.Respond(rw, r, route.Produces, route, err)
        return
    }

    res := o.Handler.Handle(Params, principal)                 // ← operation
    o.Context.Respond(rw, r, route.Produces, route, res)       // ← respond
}
```

Same primitives, same order. **Neither shape is enforced by the
runtime**: a route is just an `http.Handler`, and you can wrap or
replace it. `middleware.Builder` exists precisely to compose your
own chain on top.

## Composing extra middleware — `Builder`

```go
type Builder func(http.Handler) http.Handler
```

`Builder` is the standard `http.Handler` decorator type, aliased so
the API reads cleanly. The runtime exposes several entry points that
take one:

| Entry point                                          | Purpose                                                                 |
|------------------------------------------------------|-------------------------------------------------------------------------|
| `middleware.Serve(spec, api)`                        | Untyped, no extra middleware (uses `PassthroughBuilder`).               |
| `middleware.ServeWithBuilder(spec, api, builder)`    | Untyped, decorate the routes handler with `builder`.                    |
| `Context.APIHandler(builder, opts…)`                 | Mounts the routes plus the default Swagger UI / spec serve middleware.  |
| `Context.APIHandlerWithUI(builder, ui, opts…)`       | Same, but pick the UI flavour (`docui.SwaggerUI` / `RapiDoc` / `Redoc`).|
| `Context.RoutesHandler(builder)`                     | Just the routes — no UI middleware. Useful when you mount under your own mux. |

A typical pattern with the [`justinas/alice`](https://github.com/justinas/alice)
middleware library — log, rate-limit, then hand off to the runtime:

{{< code file="server/pipeline/main.go" lang="go" region="aliceComposition" >}}

`PassthroughBuilder` is the identity decorator if you need a place
to start.

## Failure modes by stage

| Stage              | Status | Surfaced as                                                                                                                    |
|--------------------|--------|--------------------------------------------------------------------------------------------------------------------------------|
| Router             | 404    | `errors.NotFound`                                                                                                              |
| Router             | 405    | `errors.MethodNotAllowed` (with `Allow` header)                                                                                |
| Security           | 401    | `errors.Unauthenticated` ("invalid credentials")                                                                                |
| Security           | 403    | `errors.New(403, …)` if the `Authorizer` returns a non-`errors.Error`                                                          |
| Negotiation        | 400    | malformed `Content-Type` ⇒ wrapped `errors.ParseError`                                                                          |
| Negotiation        | 415    | `errors.InvalidContentType` (no `consumes` entry matches)                                                                      |
| Negotiation        | 406    | `errors.InvalidResponseFormat` (no `produces` entry satisfies `Accept`)                                                        |
| Binding/Validation | 422    | `errors.CompositeValidationError` aggregating every parameter-level violation (does not stop on first failure)                  |
| Operation          | varies | whatever the handler returns (`error` ⇒ runs through `Context.Respond` and the API's `ServeError`)                              |

For the matching algorithm and the v0.30 parameter-honouring change
behind 415/406 outcomes, see
[standalone / content negotiation](../../standalone/content-negotiation/)
and the canonical
[tutorials / media-type selection](../../tutorials/media-types/).

## Reading values out of the request

Each stage stashes its result in the request context so downstream
middleware can read it without re-doing the work:

| Helper                                                          | Returns                                |
|-----------------------------------------------------------------|----------------------------------------|
| `middleware.MatchedRouteFrom(r) *MatchedRoute`                  | the route matched by the router         |
| `middleware.SecurityPrincipalFrom(r) any`                       | the principal returned by `Authorize`   |
| `middleware.SecurityScopesFrom(r) []string`                     | the union of scopes for the matched scheme |

Use these inside extra middleware mounted via `Builder`.
