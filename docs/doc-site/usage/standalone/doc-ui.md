---
title: Doc UIs & spec serving
weight: 30
description: |
  Stdlib-only Swagger UI, RapiDoc, Redoc and spec-serving handlers
  from the docui package.
---

[`server-middleware/docui`](https://pkg.go.dev/github.com/go-openapi/runtime/server-middleware/docui)
ships ready-to-mount `http.Handler`s that serve the three popular
OpenAPI documentation UIs and the spec document itself. Standard
library only — no template engine, no asset bundler, no transitive
OpenAPI dependency.

## Two equivalent patterns

Each UI is exposed in two shapes; pick whichever fits your wiring style.

### Direct handler wrap — `SwaggerUI(next, opts...)`

For when you already have an `http.Handler` you want to decorate.

{{< code file="standalone/docui/main.go" lang="go" region="directWrap" >}}

Requests under the configured doc path render the UI; everything else
falls through to `next`.

### Middleware factory — `UseSwaggerUI(opts...)`

For composition with other middlewares (`alice.New(...)`, your own
chain, etc.):

{{< code file="standalone/docui/main.go" lang="go" region="middlewareFactory" >}}

`Use*` returns a `func(http.Handler) http.Handler` — the standard
go-style middleware adapter.

## Available UIs

| UI                      | Direct                    | Middleware factory          |
|-------------------------|---------------------------|-----------------------------|
| Swagger UI              | `docui.SwaggerUI`         | `docui.UseSwaggerUI`        |
| Swagger UI OAuth2 cb    | `docui.SwaggerUIOAuth2Callback` | `docui.UseSwaggerUIOAuth2Callback` |
| RapiDoc                 | `docui.RapiDoc`           | `docui.UseRapiDoc`          |
| Redoc                   | `docui.Redoc`             | `docui.UseRedoc`            |

The OAuth2 callback handler is the small static page Swagger UI redirects
to after an OAuth2 authorization — mount it at the path you configure in
your OAuth provider.

## Common options

| Option                        | Purpose                                                                                  | Default                                                                                                                                                                                                              |
|-------------------------------|------------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `WithUIBasePath(string)`      | Base path the UI is served from. Slash is prepended if missing.                          | `/`                                                                                                                                                                                                                  |
| `WithUIPath(string)`          | Sub-path under the base path (final URL: `{base}/{path}`).                               | `docs`                                                                                                                                                                                                               |
| `WithUITitle(string)`         | HTML `<title>` of the rendered page.                                                     | `API documentation`                                                                                                                                                                                                  |
| `WithUIAssetsURL(string)`     | URL of the JS bundle for the UI.                                                         | Redoc → `https://cdn.jsdelivr.net/npm/redoc/bundles/redoc.standalone.js`<br/>RapiDoc → `https://unpkg.com/rapidoc/dist/rapidoc-min.js`<br/>Swagger UI → `https://unpkg.com/swagger-ui-dist/swagger-ui-bundle.js`     |
| `WithUITemplate(tpl)`         | Replace the bundled HTML template entirely (`~string` or `~[]byte`).                     | bundled minimal template                                                                                                                                                                                             |
| `WithSpecURL(string)`         | URL the UI fetches the spec from.                                                        | `/swagger.json`                                                                                                                                                                                                      |
| `WithSwaggerUIOptions(opts)`  | Pass-through for Swagger-UI-specific knobs (OAuth2 client id, layout, …).                | zero value                                                                                                                                                                                                           |

`WithUITemplate` panics at request time if the supplied template fails
to parse or execute — fail loud, not silent. Reference docs for the
templates each UI accepts:

- Redoc: <https://github.com/Redocly/redoc/blob/main/docs/deployment/html.md>
- RapiDoc: <https://github.com/rapi-doc/RapiDoc>
- Swagger UI: <https://github.com/swagger-api/swagger-ui>

## Serving the spec document — `ServeSpec` / `UseSpec`

The UIs only render — they do not host the spec document themselves.
Use the `Spec` helpers for that:

{{< code file="standalone/docui/main.go" lang="go" region="serveSpec" >}}

or as middleware:

{{< code file="standalone/docui/main.go" lang="go" region="useSpec" >}}

If you want the spec path the UIs use to stay in sync with the path
the spec is served from:

{{< code file="standalone/docui/main.go" lang="go" region="pathFromOptions" >}}

## Putting it together

A complete net/http server with no OpenAPI runtime in the picture:

{{< code file="standalone/docui/main.go" lang="go" region="puttingItTogether" >}}

Visit:

- `http://localhost:8080/docs` — Swagger UI
- `http://localhost:8080/openapi.yaml` — the spec document
- `http://localhost:8080/v1/ping` — the application
