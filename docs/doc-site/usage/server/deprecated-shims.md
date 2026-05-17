---
title: Deprecated shims
weight: 90
description: |
  Doc-UI handlers, content negotiation and the header package have
  moved to the standalone server-middleware module — this page lists
  the old entry points and shows the migration.
---

In v0.30 the server-side helpers that don't actually need any OpenAPI
machinery were extracted into the
[server-middleware](../../standalone/) module. The old entry points
in `middleware` still compile (and forward to the new ones) so
existing imports keep building, but they are tagged deprecated and
will be removed in a future major release.

This page is a cheat-sheet for the migration. New code should target
the right-hand column directly.

## Content negotiation

| Old (`middleware`)                                       | New (`server-middleware/negotiate`)                                |
|----------------------------------------------------------|--------------------------------------------------------------------|
| `middleware.NegotiateOption`                             | `negotiate.Option`                                                 |
| `middleware.NegotiateContentType(r, offers, def, opts…)` | `negotiate.ContentType(r, offers, def, opts…)`                     |
| `middleware.NegotiateContentEncoding(r, offers)`         | _deprecated, no direct replacement_ — see the [compression recipe](../../examples/middleware/compression/) |
| `middleware.WithIgnoreParameters(true)`                  | `negotiate.WithIgnoreParameters(true)`                             |

Same signatures, same semantics. The deprecated forms in
`middleware/seam.go` are thin wrappers that call straight through.

{{< code file="server/deprecatedshims/main.go" lang="go" region="negotiateBefore" >}}

{{< code file="server/deprecatedshims/main.go" lang="go" region="negotiateAfter" >}}

See [standalone / content negotiation](../../standalone/content-negotiation/)
for the full surface, including the v0.30 MIME-parameter-honouring
default.

## Header parsing

| Old (`middleware/header`)            | New (`server-middleware/negotiate/header`)             |
|--------------------------------------|--------------------------------------------------------|
| `header.AcceptSpec`                  | `header.AcceptSpec` (re-export)                        |
| `header.Copy`, `ParseList`, etc.     | same names, new path                                   |

The shim package
([`middleware/header`](https://pkg.go.dev/github.com/go-openapi/runtime/middleware/header))
re-exports everything via type aliases and forwarding functions, so
existing code is binary-compatible. Update imports when convenient.

## Doc UI handlers — `SwaggerUI`, `RapiDoc`, `Redoc`

The `middleware` shims preserve the option-struct calling convention.
The new `docui` package uses functional options and accepts
`(next http.Handler, opts ...Option)`.

| Old (`middleware`)                             | New (`server-middleware/docui`)                             |
|------------------------------------------------|-------------------------------------------------------------|
| `middleware.SwaggerUI(opts SwaggerUIOpts, next)` | `docui.SwaggerUI(next, opts ...docui.Option)`               |
| `middleware.RapiDoc(opts RapiDocOpts, next)`   | `docui.RapiDoc(next, opts ...docui.Option)`                 |
| `middleware.Redoc(opts RedocOpts, next)`       | `docui.Redoc(next, opts ...docui.Option)`                   |
| `middleware.SwaggerUIOAuth2Callback(opts, next)` | `docui.SwaggerUIOAuth2Callback(next, opts...)`              |
| `middleware.Spec(basePath, spec, next, opts…)` | `docui.ServeSpec(spec, next, docui.WithSpecPath(...))`      |

Field-to-option mapping for the `*Opts` structs:

| `*Opts` field   | `docui` option            |
|-----------------|---------------------------|
| `BasePath`      | `WithUIBasePath(s)`       |
| `Path`          | `WithUIPath(s)`           |
| `SpecURL`       | `WithSpecURL(s)`          |
| `Title`         | `WithUITitle(s)`          |
| `Template`      | `WithUITemplate(s)`       |
| `RapiDocURL` / `RedocURL` / `SwaggerURL` | `WithUIAssetsURL(s)` |
| Swagger-UI-specific knobs (`OAuthCallbackURL`, presets, favicons) | `WithSwaggerUIOptions(docui.SwaggerUIOptions{…})` |

Migration example:

{{< code file="server/deprecatedshims/main.go" lang="go" region="swaggerUIBefore" >}}

{{< code file="server/deprecatedshims/main.go" lang="go" region="swaggerUIAfter" >}}

Methods on `*Opts` types that were only used to manipulate option
structs (e.g. `SwaggerUIOpts.EnsureDefaults`) have been **removed** —
they were not load-bearing.

See [standalone / doc UIs](../../standalone/doc-ui/) for the full
options reference, the middleware-factory shape (`UseSwaggerUI`,
etc.) and a complete net/http example.

## Why the split?

Two reasons:

- **Dependency hygiene.** The doc UI and negotiation helpers don't
  need any OpenAPI machinery. Pulling them through `middleware` made
  every consumer transitively depend on `go-openapi/spec`,
  `go-openapi/loads` and `go-openapi/validate`. The standalone module
  has zero such transitive deps — handy for a service that only wants
  to serve a static spec and a Swagger UI from a vanilla `net/http`
  mux.
- **API hygiene.** The new functional options are easier to extend
  than option-struct fields, and let us keep adding knobs without
  growing struct surfaces. The deprecated shims paper over the older
  shape so old code keeps building.

The plan is to remove the shims in a future major release. Migrating
when convenient is enough — there's no urgency, but there's no reason
to keep new code on the old paths either.
