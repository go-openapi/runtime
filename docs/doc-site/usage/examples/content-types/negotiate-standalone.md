---
title: Negotiation in plain net/http
weight: 50
description: |
  Use server-middleware/negotiate from a vanilla net/http handler —
  no OpenAPI spec, no go-openapi/runtime dependency.
---

The
[`server-middleware`](../../../standalone/) module ships content
negotiation as a standalone, dependency-free package. You can drop
it into any `net/http` application — no spec, no analyzer, no
`go-openapi/runtime` import.

## Install

```sh
go get github.com/go-openapi/runtime/server-middleware
```

The full module pulls only the standard library at runtime
(testify is `_test.go`-only).

## Pick a response Content-Type

{{< code file="contenttypes/negotiatestandalone/main.go" lang="go" region="pickContentType" >}}

`ContentType` returns the most-acceptable offer per the request's
`Accept` header (q-values, specificity, position-as-tiebreaker). If
no offer is acceptable, the third argument (the *default offer*) is
returned.

## Pick a Content-Encoding

{{< code file="contenttypes/negotiatestandalone/main.go" lang="go" region="pickEncoding" >}}

`""` means "no offer is acceptable" — let your handler decide
whether to send the unencoded body or 406.

## Exercise

```sh
# JSON by preference
curl -i -H 'Accept: application/json' http://127.0.0.1:8080/pet

# XML preferred, JSON acceptable
curl -i -H 'Accept: application/xml;q=0.9, application/json;q=0.5' \
        http://127.0.0.1:8080/pet

# Both rejected → falls back to the default offer (application/json here)
curl -i -H 'Accept: text/html' http://127.0.0.1:8080/pet
```

## MIME-parameter behaviour

As of v0.30 the negotiator honours MIME parameters by default — an
`Accept` of `text/plain;charset=utf-8` does **not** match an offer of
`text/plain;charset=ascii`. Pre-v0.30 the parameters were stripped
before matching. Opt out per call to restore the old behaviour:

{{< code file="contenttypes/negotiatestandalone/main.go" lang="go" region="ignoreParameters" >}}

Full algorithm and rationale:
[standalone / content negotiation](../../../standalone/content-negotiation/).

## Adding a Swagger UI to the same server

The same module ships
[`docui`](../../../standalone/doc-ui/) — stdlib-only handlers for
Swagger UI / RapiDoc / Redoc. Combining the two gives you a small
spec-served, doc-UI-equipped HTTP server with no OpenAPI runtime
dependency at all. See [docui standalone](../../docui-standalone/)
(queued) once we write that example.
