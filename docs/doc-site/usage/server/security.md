---
title: Security schemes
weight: 30
description: |
  Server-side authenticator implementations — Basic, API key, Bearer,
  OAuth2 — and their context-aware *Ctx variants.
---

[`security`](https://pkg.go.dev/github.com/go-openapi/runtime/security)
ships ready-made `runtime.Authenticator` implementations for the four
auth flavours OpenAPI 2.0 understands. Each comes in two shapes — a
plain variant and a `*Ctx` variant that threads `context.Context`
through to your authenticate function.

## The user-supplied callback

You don't implement `Authenticator` directly — you implement a
verification callback and pass it to one of the constructors below.
The runtime handles the wire-format details (header parsing, scheme
selection, scope handling, etc.).

| Constructor                                       | Your callback signature                                                                  |
|---------------------------------------------------|------------------------------------------------------------------------------------------|
| `BasicAuth(fn)` / `BasicAuthRealm(realm, fn)`     | `func(user, password string) (principal any, err error)`                                 |
| `BasicAuthCtx(fn)` / `BasicAuthRealmCtx(…)`       | `func(ctx, user, password) (ctx, principal, err)`                                        |
| `APIKeyAuth(name, in, fn)`                        | `func(token string) (principal, err)`                                                    |
| `APIKeyAuthCtx(name, in, fn)`                     | `func(ctx, token) (ctx, principal, err)`                                                 |
| `BearerAuth(name, fn)` *(OAuth2)*                 | `func(token string, scopes []string) (principal, err)`                                   |
| `BearerAuthCtx(name, fn)` *(OAuth2)*              | `func(ctx, token, scopes) (ctx, principal, err)`                                         |

A successful callback returns the authenticated principal — typed
however your application likes. The principal is then handed to any
configured `Authorizer` and stashed in the request context (read with
`middleware.SecurityPrincipalFrom`).

## Why `*Ctx`?

Most real authenticators want request scope: a request-scoped
database handle, a tracing span, or a deadline that should propagate
into the auth lookup. The `*Ctx` constructors give your callback the
request context and let it return a (possibly enriched) context that
the runtime then attaches to the request.

{{< code file="server/security/main.go" lang="go" region="basicAuthCtx" >}}

The non-`*Ctx` variants exist for compatibility with code from before
context propagation was the norm. New code should default to `*Ctx`.

## `BasicAuth` — RFC 7617

{{< code file="server/security/main.go" lang="go" region="basicAuthSimple" >}}

`BasicAuth` reads `r.BasicAuth()` and calls your callback with the
decoded credentials. Use `BasicAuthRealm("my-realm", fn)` to set the
challenge realm advertised in `WWW-Authenticate` on failure (default:
`"Basic Realm"`).

When the request has no `Authorization` header, the authenticator
returns `(false, nil, nil)` — "scheme does not apply" — so the next
configured scheme is tried. A non-nil error from your callback is
treated as a 401.

`security.FailedBasicAuth(r)` / `FailedBasicAuthCtx(ctx)` returns the
realm name when basic auth has been attempted and failed. Useful from
custom error handlers that want to render a `WWW-Authenticate`
challenge.

## `APIKeyAuth` — header or query

{{< code file="server/security/main.go" lang="go" region="apiKeyAuthHeader" >}}

`in` must be `"header"` or `"query"` — anything else **panics** at
construction time (it is a programmer error). The callback receives
the raw token; an empty value short-circuits with
`(false, nil, nil)` so other schemes can apply.

## `BearerAuth` — OAuth2 / Bearer tokens

{{< code file="server/security/main.go" lang="go" region="bearerAuthScopes" >}}

The runtime extracts the token from, in order:

1. `Authorization: Bearer <token>`
2. The `access_token` query parameter
3. The `access_token` form field if `Content-Type` is
   `application/x-www-form-urlencoded` or `multipart/form-data`

That covers RFC 6750 §2.

`requiredScopes` is whatever the operation declared in its
`security:` block. Combine multiple security entries (per the spec)
and you'll see the union or intersection per call —
`RouteAuthenticator.AllScopes()` and `CommonScopes()` expose those if
you need to inspect them yourself.

The "scheme name" you pass (`"oauth2"` here) is recoverable from the
request via `security.OAuth2SchemeName(r)` /
`security.OAuth2SchemeNameCtx(ctx)`. That's the hook point for code
that needs to know *which* OAuth2 entry was applied (handy when a
spec declares multiple OAuth2 flows).

## Authorizer

Authentication says *who*; authorization says *may they do this?*.
Authorizer runs after a principal has been resolved.

```go
type Authorizer interface {
    Authorize(*http.Request, any) error
}
```

(see [`runtime.Authorizer`](https://pkg.go.dev/github.com/go-openapi/runtime#Authorizer))

The package ships one trivial implementation:

{{< code file="server/security/main.go" lang="go" region="registerAuthorized" >}}

Anything more interesting (RBAC, ABAC, OPA / casbin / your own…) you
write yourself. A non-nil return blocks the request:

- A return value implementing `errors.Error` is propagated as-is.
- Any other error is wrapped as `errors.New(403, err.Error())`.

The single `Authorize` call on `Context` ([core / interfaces](../../core/interfaces/#server-lifecycle--where-each-interface-fires))
runs `Authenticator` and `Authorizer` in sequence — `Authorizer` only
runs if the authenticator returned a principal.

## Composing schemes — `RouteAuthenticators`

A spec can declare multiple security requirements per operation. The
runtime turns each one into a `RouteAuthenticator` and groups them
into `RouteAuthenticators`. `RouteAuthenticators.Authenticate` walks
the list and:

- returns the first one that returned `(true, principal, nil)`;
- collects errors from any that applied but failed (last one wins for
  the response status);
- returns `AllowsAnonymous() == true` if no scheme was required —
  in that case the request proceeds without a principal.

You don't construct `RouteAuthenticators` directly — the runtime
builds them from your registered `Authenticator`s (typed APIs do this
in generated code; untyped APIs via `untyped.API.AddAuth` and
related). The grouping and short-circuit semantics are worth knowing
about when you wonder why "scheme A is rejecting and scheme B never
runs": that's by design — the first applicable scheme decides.

## Reading the principal back

Inside your operation handler, the typed signature gives you the
principal directly. From extra middleware mounted via `Builder`:

{{< code file="server/security/main.go" lang="go" region="readPrincipal" >}}

`scopes` is the `AllScopes()` of the matching `RouteAuthenticator` —
useful for audit logging that needs to record which token (or token
shape) authorised the request.
