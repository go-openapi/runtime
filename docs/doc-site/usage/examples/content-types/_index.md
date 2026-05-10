---
title: Content types & negotiation
weight: 20
description: |
  Adding new wire formats, registering vendor MIME types, streaming
  bodies, per-payload Content-Type overrides, and using the
  standalone negotiator from a vanilla net/http handler.
---

The runtime ships codecs for JSON, XML, CSV, plain text, byte streams
and YAML ([core / content-types](../../core/content-types/)). Anything
beyond that — a different format, vendor MIME types, large streaming
bodies, per-payload Content-Type overrides — is a few lines of glue.

The pages below each tackle one such glue case.

{{< children type="card" description="true" >}}
