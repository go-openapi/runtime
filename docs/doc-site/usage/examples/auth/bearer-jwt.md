---
title: Bearer + JWT
weight: 30
description: |
  OAuth2-style Bearer tokens carrying a JWT — verified locally,
  scope-checked against the operation's required scopes.
---

The runtime extracts the token from `Authorization: Bearer …` (or
the `access_token` query / form field — see
[server / security](../../../server/security/#bearerauth--oauth2--bearer-tokens)).
Your callback verifies it and decides whether the token's claimed
scopes satisfy the operation's required scopes.

## Spec

OpenAPI 2.0 only declares scopes under `type: oauth2`. Use that
declaration even if you're not running an OAuth2 dance — the runtime
treats it as "extract a Bearer token and pass me the required scopes".

```yaml
securityDefinitions:
  hasRole:
    type: oauth2
    flow: accessCode
    authorizationUrl: 'https://issuer.example.com/auth'   # documentary only
    tokenUrl:         'https://issuer.example.com/token'  # documentary only
    scopes:
      customer: regular customer
      admin:    administrative actions

security:
  - hasRole: [customer]   # default: any operation needs at least "customer"
```

## Wiring

JWT parsing is shown here via a `parseJWT` stub so the doc-examples module
does not lock you into a specific library — swap it for
[`jwt.ParseWithClaims`](https://pkg.go.dev/github.com/golang-jwt/jwt/v5#ParseWithClaims)
(or an introspection call) in your own code.

{{< code file="auth/bearerjwt/main.go" lang="go" region="wireBearerAuth" >}}

The first argument to `BearerAuth` is the **scheme name** — match the
key under `securityDefinitions`. It is recoverable from the request
via `security.OAuth2SchemeName(r)` when an operation declares more
than one OAuth2 entry.

## Token sources, in order

The runtime tries, in this order:

1. `Authorization: Bearer <token>`
2. `?access_token=…` query parameter
3. `access_token` form field if `Content-Type` is
   `application/x-www-form-urlencoded` or `multipart/form-data`

That covers RFC 6750 §2.

## Exercise

```sh
TOKEN=$(jwt -key keys/private.pem -alg RS256 sign \
    -claim 'sub=alice' -claim 'roles=["customer"]')

curl -i -H "Authorization: Bearer $TOKEN" http://127.0.0.1:8080/orders/42

# Or via query param:
curl -i "http://127.0.0.1:8080/orders/42?access_token=$TOKEN"
```

## Variations

- **Remote verification (introspection)**: replace the local
  `jwt.ParseWithClaims` with an HTTP call to your auth server's
  `/introspect` endpoint. Use `BearerAuthCtx` so the introspection
  call honours the request context.
- **OIDC / Google bearer tokens**: the
  [oauth2-access-code](../oauth2-access-code/) example shows the
  full handshake plus the token-validation callback.
- **Multiple bearer schemes**: not supported — the runtime extracts
  one token and passes it to whichever bearer authenticator applies
  for the route. The
  [composed](../composed/) example walks the standard workaround.
