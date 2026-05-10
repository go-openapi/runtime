---
title: Custom Authorizer (RBAC)
weight: 60
description: |
  Pluggable Authorizer that gates the principal for the matched
  operation — a worked role-based access control example, orthogonal
  to whichever Authenticator was used.
---

Authentication answers *who*. Authorization answers *may they do
this?* — a separate decision the runtime asks of your `Authorizer`
**after** the principal has been resolved
([core / interfaces](../../../core/interfaces/#server-lifecycle--where-each-interface-fires)).

The runtime ships one trivial authorizer (`security.Authorized()` —
always-allow). Anything more interesting you write yourself.

## A role-based authorizer

{{< code file="auth/customauthorizer/main.go" lang="go" region="rbacAuthorizer" >}}

Two things worth knowing about the return value:

- A return implementing `errors.Error` is propagated as-is (status
  code preserved).
- Any other error is wrapped as `errors.New(403, err.Error())`.

That's why the example uses `errors.New(http.StatusForbidden, …)`
rather than `fmt.Errorf` — to keep control of the status code.

## Wire it

{{< code file="auth/customauthorizer/main.go" lang="go" region="wireAuthorizer" >}}

That's it — the runtime calls `Authorize` on every authenticated
request after the authenticator has populated the principal.

## Reading the principal & scopes elsewhere

Inside extra middleware mounted via
[`middleware.Builder`](../../../server/pipeline/#composing-extra-middleware--builder),
or from a custom error handler:

{{< code file="auth/customauthorizer/main.go" lang="go" region="readPrincipal" >}}

Useful for audit logging, per-tenant rate limiting, or surfacing a
"why was this denied?" message in error responses.

## Variations

- **OPA / Casbin / your own engine**: same shape — call out to the
  policy evaluator from inside the `AuthorizerFunc`.
- **Skip authorization for some routes**: combine the ACL with a
  short-circuit on the matched route (`route.Operation.ID`,
  `route.PathPattern`, etc.) before consulting the engine.
- **Per-method body inspection**: `Authorizer` runs after
  authentication but **before** parameter binding, so the request body
  has not been consumed at this point — for body-based decisions
  ("the document the user is editing must belong to them"), do the
  check inside the operation handler, where the bound params are
  available.
