---
title: Custom middleware
weight: 30
description: |
  Composing third-party HTTP middleware around the runtime — recipes
  that wrap or extend the `http.Handler` returned by
  `middleware.Serve`.
---

The runtime pipeline (*Router → Security → Bind → Validate →
OperationHandler → Responder*) lives behind a single `http.Handler`.
Standard ecosystem middleware — compression, logging, rate-limiting,
tracing — composes around that handler the usual way. Order matters:
transport-level concerns (TLS termination, auth gating, rate limits)
typically wrap whatever middleware needs to see the final response
bytes (compression, logging), which in turn wraps the runtime
pipeline.

The pages below cover specific compositions worth pinning down.

{{< children type="card" description="true" >}}
