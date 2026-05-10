---
title: Vendor MIME types
weight: 20
description: |
  Versioning an API through vendor MIME types
  (application/vnd.acme.v1+json) — separate registrations per type,
  shared codec.
---

API versioning by vendor MIME type — `application/vnd.acme.v1+json`
and friends — is a common alternative to `/v1/` URL prefixes. The
runtime supports it, but each MIME registers as its own entry: the
`+json` structural suffix is **not** sniffed automatically.

## Spec

```yaml
consumes:
  - application/vnd.acme.v1+json
  - application/vnd.acme.v2+json
produces:
  - application/vnd.acme.v1+json
  - application/vnd.acme.v2+json
```

## Server registration

Both versions decode the same JSON wire format, so they share the
codec. They still need separate registrations:

{{< code file="contenttypes/vendortypes/main.go" lang="go" region="registerVendorTypes" >}}

`JSONConsumer()` and `JSONProducer()` are side-effect free — calling
them per registration is fine.

## Picking the version inside the handler

The matched `Content-Type` is on the request context — recover it via
`runtime.ContentType(r.Header)`:

{{< code file="contenttypes/vendortypes/main.go" lang="go" region="dispatchOnContentType" >}}

For the response side, the runtime has already chosen a producer
that matches the client's `Accept` — your handler returns a value
and the matched `Producer` writes it. If you need the response shape
to differ between versions, branch on the negotiated content-type
the same way (see `Context.ResponseFormat`).

## Matching rules — what about MIME parameters?

The
[asymmetric matching rule](../../../standalone/media-types/#the-asymmetric-matching-rule)
applies. If your spec lists a parameterised type
(`application/vnd.acme+json;version=1`), an inbound request with no
`version` parameter does **not** match. Recommend the simpler form —
parameter-distinct types are rarely worth the surprise.

For the v0.30 parameter-honouring change and the per-call opt-out,
see
[standalone / content negotiation](../../../standalone/content-negotiation/#behaviour-change-in-v030--mime-parameters-honoured).

## Adding a non-JSON vendor type

The exact same shape — register a separate Consumer/Producer per
declared MIME, even when several share the same codec:

```go
import msgpackcodec "example.com/myapp/codecs/msgpack"

api.RegisterConsumer("application/vnd.acme.v1+msgpack", msgpackcodec.Consumer())
api.RegisterProducer("application/vnd.acme.v1+msgpack", msgpackcodec.Producer())
```

See [custom codec](../custom-codec/) for the `msgpackcodec` package
itself.

## When *not* to do this

Vendor MIME types compose poorly with browser clients
(`Accept: */*` is unspecific), with caches that key on URL alone, and
with HTTP middleware that inspects the URL. URL-based versioning
(`/v1/...`) sidesteps all three. Pick vendor MIME types when the API
is server-to-server *and* you genuinely need the same URL to serve
multiple representations.
