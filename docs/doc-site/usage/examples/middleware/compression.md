---
title: Compression
weight: 10
description: |
  Adding transparent HTTP response compression (gzip, brotli, …) to
  a runtime server by wrapping the `http.Handler` returned by
  `middleware.Serve` with the CAFxX `httpcompression` adapter.
---

This example shows how to add transparent HTTP response compression
(gzip, brotli, …) to a `go-openapi/runtime` server by wrapping the
`http.Handler` returned by `middleware.Serve` with a standard
ecosystem compression middleware.

The runtime itself does not ship compression. Composition with an
external middleware is the recommended approach; this example uses
[`github.com/CAFxX/httpcompression`](https://github.com/CAFxX/httpcompression),
which covers gzip + brotli + zstd + deflate with sensible defaults
(content-type allowlist, minimum-size threshold, `Vary` / `ETag` /
`Content-Length` handling).

## The wiring

The runtime hands you an `http.Handler`. Wrap it with the
compression adapter and mount the result on the mux:

{{< code file="middleware/compression/main.go" lang="go" region="compressionWiring" >}}

`DefaultAdapter()` enables gzip + brotli with sensible defaults.
Use `Adapter(...)` for explicit codec, threshold, and content-type
control (e.g. `httpcompression.GzipCompressionLevel(6)`,
`httpcompression.MinSize(512)`,
`httpcompression.ContentTypes([]string{"application/json"}, false)`).

## Run

```sh
go run .
```

Then in another terminal:

```sh
# Plain (uncompressed) response.
curl -i http://localhost:8080/api/greeting

# Gzip-compressed response.
curl -i -H 'Accept-Encoding: gzip' http://localhost:8080/api/greeting

# Brotli-compressed response (DefaultAdapter enables brotli too).
curl -i -H 'Accept-Encoding: br' --compressed http://localhost:8080/api/greeting
```

The compressed response carries `Content-Encoding: gzip` (or `br`),
`Vary: Accept-Encoding`, and a transformed `Content-Length`. The
`go-openapi/runtime` pipeline is unchanged — the compressor sits
outside the API handler and operates on the final response bytes.

## Layering

The order of middlewares around the api handler matters:

```text
client ─► [ TLS / auth / rate-limit ] ─► [ compress ] ─► [ go-openapi api ] ─► handler
```

The compressor must wrap the api handler so it sees the complete
response body before transport. Transport-level concerns (TLS
termination, auth gating, rate limiting) typically wrap the
compressor in turn.

## Client-side

`net/http`'s default transport auto-decodes `gzip` responses, but
not `br` / `zstd` / `deflate`. Clients that need broader decoding
can wrap their `http.RoundTripper` with a decoder;
[`github.com/klauspost/compress`](https://github.com/klauspost/compress)
provides primitives suitable for that purpose. The
`go-openapi/runtime` client (`client.Runtime`) accepts a custom
transport via its configuration, so the same pattern applies.
