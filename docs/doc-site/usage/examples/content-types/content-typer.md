---
title: Per-payload Content-Type override
weight: 40
description: |
  The runtime.ContentTyper interface — declaring a payload's wire
  Content-Type from the value itself, on the client side.
---

The client normally derives the request `Content-Type` from the
operation's `consumes` list. Two cases need an override:

- a stream payload (`io.Reader` / `io.ReadCloser` set via
  `SetBodyParam`) whose actual format isn't what `consumes` defaults to
- an individual file part inside a multipart upload that has its
  own per-part Content-Type (rather than `http.DetectContentType`-sniffed)

[`runtime.ContentTyper`](https://pkg.go.dev/github.com/go-openapi/runtime#ContentTyper)
is the seam:

```go
type ContentTyper interface {
    ContentType() string
}
```

When the runtime picks up a body or file value that satisfies this
interface and `ContentType()` returns a non-empty string, **that
value wins**. An empty return is treated as "no opinion" and the
runtime falls back to its default selection.

The full algorithm — the order of precedence and how it interacts
with `consumes` and the negotiator — is in
[tutorials / media-type selection](../../../tutorials/media-types/).

## Stream payloads — naming the wire format

Use this when you're sending a binary blob whose precise format you
know, and you want the recipient (or a proxy) to see the right
header instead of `application/octet-stream`:

{{< code file="contenttypes/contenttyper/main.go" lang="go" region="streamPayload" >}}

If `imagePayload` did not implement `ContentType()`, the runtime
would use whichever entry in `op.ConsumesMediaTypes` it picked
(typically `application/octet-stream`).

## Multipart file parts — per-part Content-Type

In a multipart request, individual file values are normally typed
via `http.DetectContentType` (sniffed from the first 512 bytes).
Implementing `ContentTyper` on the file value bypasses that:

{{< code file="contenttypes/contenttyper/main.go" lang="go" region="multipartFileType" >}}

```go
// Wiring (illustrative — Params is built by the generated client):
f, _ := os.Open("manifest.json")
part := taggedFile{File: f, mime: "application/vnd.acme.manifest+json"}

// op.Params.SetFileParam("manifest", part)  ← part header carries
//                                              "Content-Type: application/vnd.acme.manifest+json"
```

Without `ContentType()` the multipart writer would sniff the bytes
and likely write `text/plain` or `application/json` — both wrong if
your downstream pipeline keys on the vendor type.

## Server-side equivalent?

There is none — server responses pick a `Producer` from the
`Accept`-negotiated `produces` entry, and the producer writes the
response. If you need to influence the response `Content-Type`
beyond what `produces` allows, use a custom `middleware.Responder`
that sets the header explicitly before delegating to the producer.

## Caveats

- `ContentTyper` is **client-side only** for body and multipart-file
  values. It is not consulted on response payloads.
- Implementing it on a value that is *not* one of those two
  (a regular struct passed as a typed body) has no effect — the
  operation's `consumes` entry wins.
- An empty `ContentType()` return is "no opinion", not "force empty
  header". The runtime falls back to its default.
