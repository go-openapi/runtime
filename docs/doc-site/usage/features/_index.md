---
title: "Features"
type: home
description: Features and compliance to internet standards.
weight: 1
---

A primer on what this runtime implementation supports, with normative
references to the standards each feature implements. Citations point at
the canonical specification rather than secondary sources.

## Client & Server

* **HTTP/1.1** and **HTTP/2** over plaintext or TLS. HTTP/2 is inherited
  transparently from Go's `net/http` stack on both client and server
  when ALPN negotiates `h2`; no runtime-specific wiring. See
  [HTTP core](#http-core) below for the supporting RFCs.
* **[Content negotiation][mdn-conneg]** on `Accept` / `Accept-Encoding`,
  honouring MIME parameters and quality values
  ([RFC 9110 §12][rfc9110-conneg], [§12.4.2 quality values][rfc9110-q]).
* **URI templating** for path-parameter expansion (Level-1 simple
  expansion only) per [RFC 6570][rfc6570]; values are percent-encoded
  per [RFC 3986 §2][rfc3986-pe].
* **Structured-suffix MIME** matching — e.g. `application/vnd.acme+json`
  falls back to the `application/json` codec
  ([RFC 6838 §4.2.8][rfc6838-suffix],
  [IANA structured-syntax-suffix registry][iana-suffix]).
* **Routing** against an analyzed OpenAPI specification.
* **Predefined codecs**:
  | Format        | Reference                                                    |
  |---------------|--------------------------------------------------------------|
  | JSON          | [RFC 8259][rfc8259]                                          |
  | XML           | [W3C XML 1.0][w3c-xml] (via Go `encoding/xml`)               |
  | CSV           | [RFC 4180][rfc4180]                                          |
  | `text/plain`  | [RFC 2046 §4.1][rfc2046-text]                                |
  | Byte stream   | `application/octet-stream` — [RFC 2046 §4.5.1][rfc2046-bin]  |
  | YAML          | [YAML 1.2][yaml-1.2] (via the `yamlpc` sub-package)          |
* **Parameter binding** for every OpenAPI parameter location:
  * **Path** parameters with URI Template Level-1 expansion
    ([RFC 6570][rfc6570]).
  * **Query** parameters.
  * **Header** parameters.
  * **Request body** decoded through the matched `Consumer`.
* **Streaming bodies**:
  * **File upload** via `multipart/form-data` ([RFC 7578][rfc7578]) or
    `application/x-www-form-urlencoded`
    ([WHATWG URL][whatwg-urlenc]).
  * Other streams via `application/octet-stream`
    ([RFC 2046 §4.5.1][rfc2046-bin]) or any custom MIME, surfaced as
    `io.Reader` / `io.Writer`.

### Trailing-slash behaviour

* **Strictly preserved by the client** — the path supplied by the
  caller is passed through verbatim.
* **Ignored by the server** — a route declared as `/pets` matches both
  `/pets` and `/pets/`.

### Optional, opt-in

* **Loosened content negotiation** (`negotiate.WithIgnoreParameters`):
  * Strip MIME parameters before matching
    (`application/json; charset=utf-8` → `application/json`).
  * Match the structured MIME suffix
    (`application/vnd.acme+json` → `application/json`).
* **Authentication bypass for dev / e2e** —
  `middleware.SetSkipAuth(true)`, available **only** in binaries built
  with the `openapi_unsafe_skipauth` Go build tag. The bypass symbol
  is absent from default builds; see
  [Security schemes / Bypassing authentication](../server/security/#bypassing-authentication-entirely--setskipauth-dev-only).

## Client

* Configurable HTTP transport, TLS / mTLS ([RFC 8446][rfc8446]), proxy
  support per [RFC 9110 §7.6.4][rfc9110-via].
* Pluggable authentication writers (see [Authentication](#authentication-schemes)).
* Built-in **OpenTelemetry** tracing ([OpenTelemetry spec][otel-spec]);
  legacy OpenTracing support remains in a sibling compatibility module.
* Debug mode — request / response dumping enabled via the
  `Runtime.Debug` field (or `Runtime.SetDebug(true)`); useful while
  iterating on a generated client.

## Server

* Composable middleware pipeline:
  *Router → Security → Bind → Validate → OperationHandler → Responder*.
* Pluggable error rendering via `api.ServeError`.
* Built-in doc-UI middleware: SwaggerUI, RapiDoc, Redoc.

## Authentication schemes

The runtime parses the standard auth headers and dispatches to
application-supplied callbacks for credential / token validation.
Token issuance, JWT signature checking, and OIDC ID-token validation
are out of scope — they belong in the callback.

* **HTTP Basic** — header parsing per [RFC 7617][rfc7617].
* **API Key** in header, query, or cookie — OpenAPI security scheme
  convention; no dedicated RFC.
* **Bearer** tokens — header parsing per [RFC 6750][rfc6750]. The
  runtime treats the bearer value as an opaque string; downstream
  parsing (JWT, opaque tokens, …) is the callback's responsibility.
* **OAuth 2.0** — the runtime exposes the same Bearer hook with the
  OAuth-2 framing ([RFC 6749][rfc6749]; [RFC 8252][rfc8252] for native
  apps). All four grant flows (authorization code, implicit, client
  credentials, password) work because the runtime sees only the
  resulting access token.

## Not supported (yet)

* **Language negotiation** — `Accept-Language` / `Content-Language`
  headers and language-tag parsing.
* **Compression** — `Accept-Encoding` / `Content-Encoding` negotiation
  and the content-coding registry (gzip, Brotli, zstd).
* **HTTP caching** — `Cache-Control` / `ETag` / `Last-Modified` /
  validators.

## Normative references

### OpenAPI specifications

* [OpenAPI v2 (Swagger 2.0)][oas2] — the dialect this runtime targets.
<!--
OpenAPI v3.x is not supported yet.
* [OpenAPI v3.0][oas30]
* [OpenAPI v3.1][oas31] — JSON-Schema-aligned successor.
-->


### HTTP core

* [RFC 9110][rfc9110] — HTTP Semantics (supersedes RFC 7230-7235).
* [RFC 9111][rfc9111] — HTTP Caching.
* [RFC 9112][rfc9112] — HTTP/1.1.
* [RFC 9113][rfc9113] — HTTP/2.
* [RFC 8446][rfc8446] — TLS 1.3 · [RFC 5246][rfc5246] — TLS 1.2.

### URIs

* [RFC 3986][rfc3986] — URI Generic Syntax.
* [RFC 6570][rfc6570] — URI Template.

### Media types

* [RFC 6838][rfc6838] — Media Type Specifications and Registration
  (structured-syntax suffixes in §4.2.8).
* [IANA structured-syntax-suffix registry][iana-suffix].
* [RFC 8259][rfc8259] — JSON.
* [W3C XML 1.0 (5th ed.)][w3c-xml].
* [RFC 4180][rfc4180] — CSV.
* [YAML 1.2][yaml-1.2].
* [RFC 7578][rfc7578] — `multipart/form-data`.
* [WHATWG URL — `application/x-www-form-urlencoded`][whatwg-urlenc].

### Authentication

* [RFC 7617][rfc7617] — HTTP Basic.
* [RFC 6749][rfc6749] — OAuth 2.0 Authorization Framework.
* [RFC 6750][rfc6750] — OAuth 2.0 Bearer Token Usage.
* [RFC 8252][rfc8252] — OAuth 2.0 for Native Apps.
<!--
JWT is not directly used by the runtime today (Bearer tokens are
opaque strings). Re-enable when first-class JWT support lands.
* [RFC 7519][rfc7519] — JSON Web Token (JWT).
-->

### Tracing

* [OpenTelemetry specification][otel-spec].
<!--
W3C Trace Context is consumed transitively via OpenTelemetry's default
propagator — list it explicitly only once we surface a runtime-level
knob for it.
* [W3C Trace Context][w3c-trace-context].
-->

<!--
Language, encoding, caching — not yet implemented. Kept as a hidden
reference list to revive when first-class support lands.

### Language, encoding, caching (not yet implemented)

* [RFC 9110 §8.5 — Accept-Language][rfc9110-lang].
* [RFC 9110 §8.4 — Accept-Encoding][rfc9110-enc].
* [RFC 9111][rfc9111] — HTTP Caching.
* [BCP 47][bcp47] / [RFC 5646][rfc5646] — Language Tags.
* Content codings: [gzip (RFC 1952)][rfc1952],
  [Brotli (RFC 7932)][rfc7932], [zstd (RFC 8478)][rfc8478].
-->


[mdn-conneg]: https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/Content_negotiation

[oas2]:  https://swagger.io/specification/v2/
<!--
[oas30]: https://spec.openapis.org/oas/v3.0.4
[oas31]: https://spec.openapis.org/oas/v3.1.1
-->


[rfc9110]:         https://www.rfc-editor.org/rfc/rfc9110
[rfc9110-conneg]:  https://www.rfc-editor.org/rfc/rfc9110#section-12
[rfc9110-q]:       https://www.rfc-editor.org/rfc/rfc9110#section-12.4.2
[rfc9110-via]:     https://www.rfc-editor.org/rfc/rfc9110#section-7.6.4
[rfc9110-lang]:    https://www.rfc-editor.org/rfc/rfc9110#section-8.5
[rfc9110-enc]:     https://www.rfc-editor.org/rfc/rfc9110#section-8.4
[rfc9111]:         https://www.rfc-editor.org/rfc/rfc9111
[rfc9112]:         https://www.rfc-editor.org/rfc/rfc9112
[rfc9113]:         https://www.rfc-editor.org/rfc/rfc9113
[rfc8446]:         https://www.rfc-editor.org/rfc/rfc8446
[rfc5246]:         https://www.rfc-editor.org/rfc/rfc5246

[rfc3986]:    https://www.rfc-editor.org/rfc/rfc3986
[rfc3986-pe]: https://www.rfc-editor.org/rfc/rfc3986#section-2
[rfc6570]:    https://www.rfc-editor.org/rfc/rfc6570

[rfc6838]:        https://www.rfc-editor.org/rfc/rfc6838
[rfc6838-suffix]: https://www.rfc-editor.org/rfc/rfc6838#section-4.2.8
[iana-suffix]:    https://www.iana.org/assignments/media-type-structured-suffix/media-type-structured-suffix.xhtml
[rfc8259]:        https://www.rfc-editor.org/rfc/rfc8259
[w3c-xml]:        https://www.w3.org/TR/xml/
[rfc4180]:        https://www.rfc-editor.org/rfc/rfc4180
[yaml-1.2]:       https://yaml.org/spec/1.2.2/
[rfc2046-text]:   https://www.rfc-editor.org/rfc/rfc2046#section-4.1
[rfc2046-bin]:    https://www.rfc-editor.org/rfc/rfc2046#section-4.5.1
[rfc7578]:        https://www.rfc-editor.org/rfc/rfc7578
[whatwg-urlenc]:  https://url.spec.whatwg.org/#application/x-www-form-urlencoded

[rfc7617]: https://www.rfc-editor.org/rfc/rfc7617
[rfc6749]: https://www.rfc-editor.org/rfc/rfc6749
[rfc6750]: https://www.rfc-editor.org/rfc/rfc6750
[rfc7519]: https://www.rfc-editor.org/rfc/rfc7519
[rfc8252]: https://www.rfc-editor.org/rfc/rfc8252

[otel-spec]:         https://opentelemetry.io/docs/specs/otel/
[w3c-trace-context]: https://www.w3.org/TR/trace-context/

[bcp47]:   https://www.rfc-editor.org/info/bcp47
[rfc5646]: https://www.rfc-editor.org/rfc/rfc5646
[rfc1952]: https://www.rfc-editor.org/rfc/rfc1952
[rfc7932]: https://www.rfc-editor.org/rfc/rfc7932
[rfc8478]: https://www.rfc-editor.org/rfc/rfc8478
