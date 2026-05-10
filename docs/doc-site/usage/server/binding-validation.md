---
title: Parameter binding & validation
weight: 20
description: |
  How path, query, header and body parameters are bound to Go values
  and validated against the spec.
---

The two stages combined as
[`Context.BindValidRequest`](https://pkg.go.dev/github.com/go-openapi/runtime/middleware#Context.BindValidRequest)
turn the incoming `*http.Request` into a populated parameter struct
and surface every spec-level violation in a single response.

## What gets bound, in what order

`BindValidRequest` runs four sub-steps. Any non-recoverable error
short-circuits before the binder runs; otherwise binder-level errors
are aggregated alongside negotiation errors:

1. **Content-Type validation** — `runtime.HasBody(r)` early-outs for
   bodyless requests; otherwise `runtime.ContentType(r.Header)` parses
   the header (a malformed value is a 400) and `validateContentType`
   matches it against the operation's `consumes` (no match ⇒ 415,
   match ⇒ pick the registered `Consumer`; missing `Consumer` ⇒ 500).
2. **Response format selection** — `negotiate.ContentType(r, route.Produces, …)`
   picks the offer that best satisfies `Accept`; `""` ⇒ 406
   (`errors.InvalidResponseFormat`).
3. **Parameter binding** — for each declared parameter, the binder
   reads the right place (path / query / header / formData / body),
   converts the string(s) to the target Go type and applies any
   default declared in the spec.
4. **Per-parameter validation** — the spec's declarative rules
   (`required`, `pattern`, `minLength`, `enum`, `format`, …) plus any
   `Validatable` / `ContextValidatable` your model implements.

All errors collected during binding and validation are returned as
one `errors.CompositeValidationError`. The validator does **not**
stop on first failure — a request with three problems produces three
entries, so callers learn about everything in one round-trip.

## Where each parameter `in:` reads from

| `in:`        | Source                                         | Notes                                                                                |
|--------------|------------------------------------------------|--------------------------------------------------------------------------------------|
| `path`       | the matched route's `RouteParams`              | Names come from the `{placeholder}` segments. Required by definition (no default).   |
| `query`      | `r.URL.Query()`                                | Multi-valued: see `collectionFormat` (`csv`, `ssv`, `tsv`, `pipes`, `multi`).         |
| `header`     | `r.Header`                                     | Multi-valued via the same `collectionFormat`s; `multi` repeats the header name.       |
| `formData`   | `r.PostForm` for `application/x-www-form-urlencoded`<br/>or `r.MultipartForm` for `multipart/form-data` | File parts come back as `runtime.File`.                                              |
| `body`       | `r.Body`, decoded via the chosen `Consumer`    | Validation runs against the resulting Go value, including any `Validatable` hook.    |

The binder is reflection-based for the untyped path; generated code
uses the same primitives by calling
`Context.BindValidRequest(r, route, &Params)` where `&Params` is the
generated parameter struct.

## Validation layers

Two layers compose. They are not alternatives.

```text
1. Spec-driven validation
   ├─ required, pattern, minLength/maxLength
   ├─ minimum/maximum, multipleOf, exclusive bounds
   ├─ enum, format (date-time, uuid, email, …)
   └─ items / minItems / maxItems / uniqueItems

2. Validatable / ContextValidatable
   ├─ Validate(strfmt.Registry) error                       (sync)
   └─ ContextValidate(ctx, strfmt.Registry) error           (request-scoped)
```

See [core / validation](../../core/validation/) for the full picture
of the hooks; `BindValidRequest` is the call site.

## Where this fits in the pipeline

Conventionally **after** security and **before** the operation
handler — see [pipeline](../pipeline/) for the diagram and the
rationale (failed auth short-circuits with 401 before paying the cost
of binding/validation).

## Disabling spec-driven parameter validation

If you need to bypass the `parameters` block entirely (typically for
test harnesses or proxy layers that re-validate downstream),
`Context.SetIgnoreParameters(true)` skips spec-driven parameter
validation while leaving the rest of the pipeline intact:

{{< code file="server/binding/main.go" lang="go" region="ignoreParameters" >}}

`Validatable` / `ContextValidatable` hooks on the model still run.

## Reading the bound parameters from extra middleware

Bound parameters are cached in the request context. From middleware
mounted via `Builder` you can re-fetch them without re-binding:

{{< code file="server/binding/main.go" lang="go" region="readMatchedRoute" >}}

`MatchedRouteFrom` plus `SecurityPrincipalFrom` and
`SecurityScopesFrom` cover the most common middleware needs (audit
logging, per-tenant rate limiting, …).
