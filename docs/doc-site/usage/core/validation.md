---
title: Validation hooks
weight: 30
description: |
  Validatable and ContextValidatable interfaces, when the runtime
  invokes them, and how they interact with spec-based validation.
---

OpenAPI specifies most validation declaratively (required fields, pattern,
min/max, enum, etc.). go-swagger turns those rules into code on the
generated model types via two interfaces:

```go
type Validatable interface {
    Validate(strfmt.Registry) error
}

type ContextValidatable interface {
    ContextValidate(context.Context, strfmt.Registry) error
}
```

Both live in the root [`runtime`](https://pkg.go.dev/github.com/go-openapi/runtime)
package — see [`runtime.Validatable`](https://pkg.go.dev/github.com/go-openapi/runtime#Validatable)
and [`runtime.ContextValidatable`](https://pkg.go.dev/github.com/go-openapi/runtime#ContextValidatable)
for the authoritative definitions. The `strfmt.Registry` argument carries
the active string-format registry (date-time, UUID, …) so format-aware
validation has access to it.

`ContextValidatable` is the context-aware version; it should be preferred
in new code because some validations (read-only / write-only flags,
async-driven cross-field checks) genuinely need request scope.

## When the runtime calls them

Server-side, validation runs as part of `Context.BindValidRequest`,
which fires **after** [security](../interfaces/#server-lifecycle--where-each-interface-fires)
and **just after** parameter binding:

{{< mermaid align="center" zoom="true" >}}
flowchart TD
    sec["Security · Context.Authorize<br/>Authenticator → Authorizer"]
    bind["Binder<br/>Consumer decodes body into the parameter struct"]
    val["Validator (per parameter)<br/>1. spec-driven validation (required, pattern, …)<br/>2. if Validatable: Validate(formats)<br/>3. if ContextValidatable: ContextValidate(ctx, formats)"]
    err{{"errors.CompositeValidationError<br/>aggregates every parameter-level violation<br/>(does not stop on first failure)"}}

    sec --> bind --> val
    val -. on error .-> err
{{< /mermaid >}}

Two consequences worth being aware of:

- **Multiple errors per parameter set.** A request with three invalid
  fields produces a `CompositeValidationError` containing three entries,
  not a single one.
- **Both layers run.** Implementing `Validatable` does not turn off
  spec-driven validation; the two layers compose. Use `Validatable` for
  rules the spec cannot express (cross-field invariants, business rules).

Client-side, generated request models implement the same interfaces, and
the generated `Validate` method runs before the body is serialised — a
malformed payload fails locally instead of producing a server-side 422.

## Custom validation in your own types

Most users never write these by hand — they fall out of `swagger generate`.
But for hand-rolled types you can add cross-field checks like this:

{{< code file="core/validation/main.go" lang="go" region="dateRangeValidate" >}}

For checks that depend on the request:

{{< code file="core/validation/main.go" lang="go" region="contextValidate" >}}

## Strfmt registry

Both methods take a `strfmt.Registry`, which is how the runtime carries
named formats (`date-time`, `uuid`, `email`, …) into the validator. You
rarely build one by hand — the server's `*Context` and the client `Runtime`
each carry one and pass it down. To register a custom format
(`x-go-type` style), call `strfmt.Default.Add(...)` once at startup; the
default registry is what both sides use unless overridden.
