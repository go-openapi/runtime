---
title: Custom codec (MessagePack)
weight: 10
description: |
  Register a Consumer and Producer for a wire format the runtime does
  not ship — using MessagePack as the worked example.
---

`Consumer` and `Producer` are functions; adding a codec for a new
wire format is just writing two of them and registering them under
the right MIME type. This page uses
[`github.com/vmihailenco/msgpack/v5`](https://pkg.go.dev/github.com/vmihailenco/msgpack/v5)
as the worked example because it's the most widely-used Go MessagePack
implementation; any third-party codec works the same way.

## Pick a Content-Type

MessagePack has no IANA-registered MIME. Two conventions are common:

- `application/x-msgpack` (older `x-` style)
- `application/msgpack`   (newer)

Pick one and stick to it across spec, server registration and client
expectation. The examples below use `application/x-msgpack`.

## The Consumer + Producer pair

{{< code file="contenttypes/customcodec/main.go" lang="go" region="consumerProducerPair" >}}

Two-line implementations are typical; the runtime never inspects
codec internals. Anything more sophisticated (configurable encoder
options, format-specific error wrapping) goes inside the closure.

## Register on the server

Spec — declare the new MIME under `consumes` / `produces`:

```yaml
consumes:
  - application/json
  - application/x-msgpack
produces:
  - application/json
  - application/x-msgpack
```

Wire it up:

{{< code file="contenttypes/customcodec/main.go" lang="go" region="registerOnServer" >}}

The runtime now picks MessagePack whenever the inbound `Content-Type`
matches and the route lists `application/x-msgpack` under `consumes`,
or `Accept: application/x-msgpack` selects it from `produces`.

## Register on the client

{{< code file="contenttypes/customcodec/main.go" lang="go" region="registerOnClient" >}}

For an individual call, set the operation's content-type lists:

{{< code file="contenttypes/customcodec/main.go" lang="go" region="operationMediaTypes" >}}

## Exercise

```sh
# Server happily decodes a MessagePack body
curl -i -H 'Content-Type: application/x-msgpack' \
        --data-binary @payload.msgpack \
        http://127.0.0.1:8080/v1/items

# And produces MessagePack on request
curl -i -H 'Accept: application/x-msgpack' \
        http://127.0.0.1:8080/v1/items/42
```

A request with `Content-Type` outside the operation's `consumes`
list yields **415 Unsupported Media Type**; an `Accept` outside
`produces` yields **406 Not Acceptable**. See
[server / pipeline](../../../server/pipeline/#failure-modes-by-stage)
for the full failure-mode mapping.

## Variations

- **Vendor MIME types** (`application/vnd.acme.v1+msgpack`) need
  separate registrations even when they delegate to the same codec —
  see [vendor types](../vendor-types/).
- **Streaming bodies**: `Consumer` / `Producer` get an `io.Reader` /
  `io.Writer` directly, so streaming codecs work the same way. The
  [streaming bodies](../streaming-bodies/) page covers raw-byte
  payloads and the `ClosesStream` option.
