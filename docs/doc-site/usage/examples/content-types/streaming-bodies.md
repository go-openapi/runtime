---
title: Streaming bodies
weight: 30
description: |
  ByteStreamConsumer and ByteStreamProducer for large up- and
  downloads — without buffering the whole payload in memory.
---

For payloads that are not naturally a single Go value — large file
downloads, log streams, raw binary uploads — `runtime.ByteStreamConsumer`
and `runtime.ByteStreamProducer` give you `io.Reader` / `io.Writer`
access without the runtime decoding into a typed model.

## Server — streaming a download

Spec:

```yaml
paths:
  /backups/{id}:
    get:
      operationId: GetBackup
      produces:
        - application/octet-stream
      responses:
        '200':
          description: backup blob
          schema:
            type: string
            format: binary
```

Wiring:

{{< code file="contenttypes/streamingbodies/main.go" lang="go" region="serverDownload" >}}

`Produce` accepts an `io.Reader` (yes, despite the name): the
default `ByteStreamProducer` copies bytes through. For typed bodies
the runtime would marshal first; here you stay in raw-byte territory
end to end.

## Server — streaming an upload

Spec:

```yaml
paths:
  /backups:
    post:
      operationId: PutBackup
      consumes:
        - application/octet-stream
      parameters:
        - in: body
          name: blob
          schema:
            type: string
            format: binary
      responses: {…}
```

Wiring:

{{< code file="contenttypes/streamingbodies/main.go" lang="go" region="consumerWithCloses" >}}

`ClosesStream` is the option to use when the consumer should
`Close()` the underlying reader after consumption. Default is *not*
to close — useful when you want to inspect the same body twice or
the caller manages the lifetime explicitly.

The bound parameter is an `io.ReadCloser`; stream straight to disk:

{{< code file="contenttypes/streamingbodies/main.go" lang="go" region="serverUpload" >}}

## Client — sending and receiving streams

Build a client request whose body is an `io.Reader` (or
`runtime.NamedReadCloser` if you also want a filename for the
`Content-Disposition`):

{{< code file="contenttypes/streamingbodies/main.go" lang="go" region="clientStream" >}}

For multipart uploads with file parts and form fields, the shape
differs — see [client multipart](../../client-multipart/) (queued).

## Choosing between ByteStream and a typed Consumer

Use `ByteStreamConsumer` / `Producer` when:

- the payload is genuinely opaque bytes (downloads, uploads of
  binary blobs, logs)
- the size could exceed RAM — buffered codecs would OOM
- you want to forward the body to another service without
  re-encoding

Use a typed Consumer/Producer (JSON, XML, …, [custom codec](../custom-codec/))
when the payload is a structured value the operation handler needs
to inspect.

The two are not mutually exclusive — a single API can route some
operations to streams and others to typed payloads via
operation-level `consumes` / `produces`.
