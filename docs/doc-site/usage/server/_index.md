---
title: Server
weight: 30
description: |
  Server-side request lifecycle — routing, parameter binding,
  validation, security and operation execution.
---

The `middleware` package wires an analyzed OpenAPI spec into a working HTTP
handler. Requests flow through a chain of stages — by default
`Router → Security → ContentType/Accept → Binder → Validator → OperationExecutor → Responder`
— composable via `middleware.Builder`. Generated typed APIs assemble an
equivalent chain explicitly per operation; either way the runtime does
not enforce a single fixed pipeline.

{{< children type="card" description="true" >}}
