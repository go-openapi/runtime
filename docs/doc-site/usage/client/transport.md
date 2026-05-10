---
title: Transport
weight: 10
description: |
  Configuring client.Runtime — TLS, timeouts, proxy, keepalive and
  the underlying http.Client.
---

[`client.Runtime`](https://pkg.go.dev/github.com/go-openapi/runtime/client#Runtime)
wraps a `*http.Client` plus the wire-format codecs needed to call an
OpenAPI-described API. This page covers the knobs that shape the
underlying HTTP behaviour; auth, tracing and request submission live
on their own pages.

## Constructors

```go
func New(host, basePath string, schemes []string) *Runtime
func NewWithClient(host, basePath string, schemes []string, client *http.Client) *Runtime
```

See [`client.New`](https://pkg.go.dev/github.com/go-openapi/runtime/client#New)
and [`client.NewWithClient`](https://pkg.go.dev/github.com/go-openapi/runtime/client#NewWithClient)
for the authoritative signatures.

`New` builds a runtime against `http.DefaultTransport`. `NewWithClient`
takes an explicit `*http.Client` — use it when you need a non-default
transport (custom TLS, a proxy, an instrumented round-tripper, etc.)
or want to share a client across runtimes.

`schemes` lists allowed URL schemes (`"https"`, `"http"`); the runtime
picks one when building a request, preferring HTTPS.

## What `New` sets up for you

| Field                      | Default                                                                                   |
|----------------------------|-------------------------------------------------------------------------------------------|
| `DefaultMediaType`         | `application/json` (`runtime.JSONMime`)                                                   |
| `Consumers` / `Producers`  | JSON, XML, YAML, plain text, HTML, CSV and `application/octet-stream` byte stream codecs. |
| `Transport`                | `http.DefaultTransport`                                                                   |
| `Context`                  | `context.Background()` (legacy field — see [requests](../requests/))                      |
| `Debug`                    | enabled if `SWAGGER_DEBUG` or `DEBUG` env var is set                                      |

You can replace any of these after construction. Example: register a
custom codec for a vendor JSON content type that the client will
encounter on responses.

{{< code file="client/transport/main.go" lang="go" region="registerVendorCodec" >}}

## TLS — `TLSClientAuth` and `TLSClientOptions`

For mutual TLS, custom CAs or certificate pinning, build a
`*tls.Config` via
[`TLSClientAuth`](https://pkg.go.dev/github.com/go-openapi/runtime/client#TLSClientAuth):

{{< code file="client/transport/main.go" lang="go" region="setupMutualTLS" >}}

Option highlights (the full struct is in the godoc):

| Group                | Fields                                                                                              |
|----------------------|-----------------------------------------------------------------------------------------------------|
| Client cert (paths)  | `Certificate`, `Key`                                                                                |
| Client cert (loaded) | `LoadedCertificate`, `LoadedKey`                                                                    |
| Server CAs           | `CA`, `LoadedCA`, `LoadedCAPool` (combined with each other; otherwise the system pool is used)      |
| Hostname / verify    | `ServerName`, `InsecureSkipVerify` (ignored when `ServerName` is set), `VerifyPeerCertificate`, `VerifyConnection` |
| Resumption           | `SessionTicketsDisabled`, `ClientSessionCache`                                                      |

`TLSClientAuth` always returns a config with `MinVersion = TLS 1.2`.

## Timeouts

Two layers of timeout apply:

1. **Per-request timeout** — set via the operation's
   `Params.SetTimeout(d)` (any generated parameter type implements this).
   This becomes the deadline of the request `context.Context` derived
   inside `BuildHTTPContext` (see [requests](../requests/)).
2. **HTTP client timeout** — set on the `*http.Client` you pass to
   `NewWithClient`. This is the standard `Client.Timeout` field; it
   applies regardless of the per-request value.

There is also a package-level
`DefaultTimeout = 30 * time.Second`. It is **not** wired up
automatically; it exists for callers building their own `*http.Client`
that want to use the same default the runtime advertises.

{{< code file="client/transport/main.go" lang="go" region="timeoutClient" >}}

## Proxy

Proxy configuration lives on the underlying `*http.Transport`, not on
`Runtime`. Two common patterns:

1. Honour `HTTPS_PROXY` / `HTTP_PROXY` (default behaviour anyway):

{{< code file="client/transport/main.go" lang="go" region="proxyFromEnv" >}}

2. Force a specific proxy:

{{< code file="client/transport/main.go" lang="go" region="proxyExplicit" >}}

## Keepalive — `EnableConnectionReuse`

Some servers never close the response body, which prevents Go from
reusing the underlying TCP connection. `EnableConnectionReuse`
installs a transport middleware that drains the unread body on
`Close()` so the connection can return to the pool:

{{< code file="client/transport/main.go" lang="go" region="enableConnectionReuse" >}}

This is **not** enabled by default because for some servers the
response stream never completes and draining would block forever.
Turn it on when you've confirmed the server you're talking to does
the right thing.

## Debug logging

Two ways to enable wire-level dumps of requests and responses (both
go through `httputil.DumpRequest` / `DumpResponse`):

- Set the `SWAGGER_DEBUG` (or `DEBUG`) environment variable before the
  process starts. `client.New` picks this up.
- Call `rt.SetDebug(true)` at runtime.

`rt.SetLogger(myLogger)` swaps the destination away from the default
standard-library logger.

For most production debugging you'll get more value out of the
[OpenTelemetry tracing](../tracing/) than from raw dumps.
