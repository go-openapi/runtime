---
title: Examples
weight: 50
description: |
  Runnable snippets covering common runtime usage scenarios — server
  assembly, client setup, custom middleware and authentication.
---

Each page below is a self-contained snippet using the **untyped** API
setup so the runtime primitives are visible. Typed (go-swagger
generated) servers call exactly the same primitives — the wiring
file is just generated for you. Where a topic has more material than
fits on a page (like authentication), it gets its own subsection.

For a fully runnable copy of any of these patterns, the
[`go-swagger/examples`](https://github.com/go-swagger/examples)
sibling repo has end-to-end programs you can clone and run.

## Subsections

{{< children type="card" description="true" depth=1 >}}
