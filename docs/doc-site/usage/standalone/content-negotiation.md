---
title: Content negotiation
weight: 20
description: |
  Accept and Accept-Encoding selection via the negotiate package,
  including the new MIME-parameter-aware default.
---

[`server-middleware/negotiate`](https://pkg.go.dev/github.com/go-openapi/runtime/server-middleware/negotiate)
sits on top of [`mediatype`](../media-types/) and exposes two
single-purpose helpers — one for `Accept`, one for `Accept-Encoding`.

## `ContentType` — pick a response media type

```go
func ContentType(
    r *http.Request,
    offers []string,
    defaultOffer string,
    opts ...Option,
) string
```

Returns the offer most acceptable to the request's `Accept` header. If
two offers match with equal weight, the more specific offer wins
(`text/*` trumps `*/*`; `type/subtype` trumps `type/*`); after that the
earlier entry in `offers` wins. If no offer is acceptable,
`defaultOffer` is returned.

{{< code file="standalone/contentnegotiation/main.go" lang="go" region="pickContentType" >}}

When `Accept` is absent entirely, the **first offer** is returned
unchanged.

### Behaviour change in v0.30 — MIME parameters honoured

Pre-v0.30 the negotiator stripped MIME-type parameters before matching:
an `Accept` of `text/plain;charset=utf-8` matched an offer of
`text/plain;charset=ascii` (the charset was thrown away). That was
expedient but wrong; v0.30 honours parameters by default:

- `Accept: text/plain;charset=utf-8` matches an offer of bare
  `text/plain` (offer carries no constraint — receiver-side params,
  [asymmetric rule](../media-types/#the-asymmetric-matching-rule)).
- `Accept: text/plain;charset=utf-8` does **not** match an offer of
  `text/plain;charset=ascii` (charset values disagree).

If your producers and `Accept` clients use mismatched charset or
version params that you treat as informational, opt out per call —

{{< code file="standalone/contentnegotiation/main.go" lang="go" region="ignoreParameters" >}}

— or server-wide via the runtime's `middleware.Context`:

{{< code file="standalone/contentnegotiation/main.go" lang="go" region="serverWideIgnoreParameters" >}}

## `Accept-Encoding` — not handled here

`negotiate.ContentEncoding` is **deprecated**. The runtime does not ship
response compression, and surfacing a half-feature negotiator without a
matching encoder leads to subtle correctness traps (no `Vary`,
no `Content-Length` rewrite, no minimum-size guard). Use a real
compression middleware at the `http.Handler` level — see the
[compression recipe](../../examples/middleware/compression/) for a worked
example using
[`CAFxX/httpcompression`](https://github.com/CAFxX/httpcompression).

## Direct header parsing

If you only need raw header parsing without the typed `MediaType`
layer (for example when implementing a different selection rule), drop
down to
[`negotiate/header`](https://pkg.go.dev/github.com/go-openapi/runtime/server-middleware/negotiate/header):

{{< code file="standalone/contentnegotiation/main.go" lang="go" region="parseAcceptHeader" >}}

## Where it sits in the runtime pipeline

The full server pipeline calls `ContentType` (and the matching
`Content-Type` validation through `mediatype.MatchFirst`) inside
`Context.BindValidRequest`; see
[core / interfaces](../../core/interfaces/#server-lifecycle--where-each-interface-fires).
The standalone module exposes the same primitives so you can drive
negotiation from any `net/http` handler, with or without an OpenAPI
spec in the picture.
