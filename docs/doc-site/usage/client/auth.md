---
title: Authentication
weight: 20
description: |
  Attaching auth information to outgoing requests — Basic, API key,
  Bearer and OAuth2.
---

Client-side authentication is a pure encoding concern: take some
credentials, write the right header / query parameter on the outbound
request. It is decoupled from the server-side `Authenticator` /
`Authorizer` interfaces ([core / interfaces](../../core/interfaces/))
— those answer "is this request allowed?", these answer "how do I
sign it?".

## The interface — `ClientAuthInfoWriter`

```go
package runtime

type ClientAuthInfoWriter interface {
    AuthenticateRequest(ClientRequest, strfmt.Registry) error
}

type ClientAuthInfoWriterFunc func(ClientRequest, strfmt.Registry) error
```

See [`runtime.ClientAuthInfoWriter`](https://pkg.go.dev/github.com/go-openapi/runtime#ClientAuthInfoWriter)
for the authoritative definition. Anything with that signature can be
used as auth. The `ClientRequest` argument exposes `SetHeaderParam`,
`SetQueryParam`, `SetBodyParam` — i.e. the same surface generated
parameter types use to encode themselves.

## Where to attach it

Two places, with predictable precedence:

{{< code file="client/auth/main.go" lang="go" region="attachAuth" >}}

The runtime calls the operation's `AuthInfo` if set, otherwise the
runtime's `DefaultAuthentication`. Either may be nil for unauthenticated
endpoints.

## Built-in helpers

All four return a ready-to-use `ClientAuthInfoWriter`.

### `BasicAuth(user, password)` — RFC 7617

{{< code file="client/auth/main.go" lang="go" region="basicAuth" >}}

Sets `Authorization: Basic <base64(user:password)>`.

### `APIKeyAuth(name, in, value)` — RFC-undefined but ubiquitous

{{< code file="client/auth/main.go" lang="go" region="apiKeyAuth" >}}

`in` must be `"header"` or `"query"`. Anything else returns nil — at
which point you'll silently send the request unauthenticated, so check
your spelling.

### `BearerToken(token)` — RFC 6750 OAuth2 access tokens

{{< code file="client/auth/main.go" lang="go" region="bearerAuth" >}}

Sets `Authorization: Bearer <token>`. For OAuth2 client flows that
need to acquire and refresh the token, build the writer around an
`oauth2.TokenSource` from `golang.org/x/oauth2` and re-attach it on
every call (or use a custom writer that calls `Token()`).

### `Compose(auths…)` — combine multiple writers

For APIs that require more than one credential header on the same
request — say an API key plus a bearer token — chain them:

{{< code file="client/auth/main.go" lang="go" region="composeAuth" >}}

Nil writers in the list are skipped silently. The first non-nil
writer that returns an error short-circuits the chain.

### `PassThroughAuth` — explicit "no auth"

A no-op writer. Use it when the operation requires *some* writer
(for instance because it's defined as `security: [[]]` in the spec)
but no actual credential should be attached.

{{< code file="client/auth/main.go" lang="go" region="passThroughAuth" >}}

## Writing your own

A common case: an HMAC-signed request that needs to compute the
signature over the body. Implement `ClientAuthInfoWriter` directly:

{{< code file="client/auth/main.go" lang="go" region="hmacSignature" >}}

The runtime calls `AuthenticateRequest` after the operation's
parameters have been bound but before the request is sent — so
`r.GetBody()` returns the encoded body for buffered payloads. For
streaming bodies (multipart, raw streams) the runtime arranges a
body-copy closure so the signer sees the bytes that will go on the
wire; see `BuildHTTPContext` in
[`client/internal/request`](https://pkg.go.dev/github.com/go-openapi/runtime/client/internal/request)
for the gory details.
