---
title: Composed schemes (AND / OR)
weight: 40
description: |
  Multiple security schemes per operation — AND inside an entry,
  OR between entries — with a single principal type.
---

Mirrors the
[`go-swagger/examples/composed-auth`](https://github.com/go-swagger/examples/tree/master/composed-auth)
example, condensed. That sibling repo has the full runnable code,
the JWT helpers, the keypair-generation script and a curl exerciser.

## The composition rule

Inside one `security` list entry, all schemes must succeed (**AND**).
Between entries, any successful entry wins (**OR**). The runtime
stops at the first entry that authenticates.

```yaml
security:
  # OR
  - isRegistered: []                   # entry 1: AND of one scheme
    hasRole: [customer]
  - isReseller: []                     # entry 2: AND of two schemes
    hasRole: [inventoryManager]
  - isResellerQuery: []                # entry 3: alternative carrier
    hasRole: [inventoryManager]
```

That reads as: *(registered AND customer-scoped)* **OR**
*(reseller-by-header AND inventory-manager-scoped)* **OR**
*(reseller-by-query AND inventory-manager-scoped)*.

## Spec sketch

```yaml
securityDefinitions:
  isRegistered:                      # Authorization: Basic …
    type: basic
  isReseller:                        # X-Custom-Key: <jwt>
    type: apiKey
    in: header
    name: X-Custom-Key
  isResellerQuery:                   # ?CustomKeyAsQuery=<jwt>
    type: apiKey
    in: query
    name: CustomKeyAsQuery
  hasRole:                           # Bearer + scopes
    type: oauth2
    flow: accessCode
    authorizationUrl: 'https://example.com/auth'   # documentary
    tokenUrl:         'https://example.com/token'  # documentary
    scopes:
      customer:         regular customer
      inventoryManager: reseller managing inventory
```

## Wiring

{{< code file="auth/composed/main.go" lang="go" region="wireComposedAuth" >}}

The callbacks (`authenticateBasic`, `verifyResellerToken`,
`verifyBearerWithScopes`) each return the same principal type — the
runtime hands the principal of the *winning* entry to the operation
handler, regardless of which schemes participated.

## One principal, many origins

A common consequence of OR composition is that you can't tell from
the operation handler alone *which* path authorized the call. Two
patterns:

- **Annotate inside the callback**: stash the auth flavour on the
  principal struct (`principal.Source = "basic"` etc.) before
  returning it.
- **Read it back from the request context**: for OAuth2 entries, use
  [`security.OAuth2SchemeName(r)`](https://pkg.go.dev/github.com/go-openapi/runtime/security#OAuth2SchemeName)
  to recover the matched scheme name. For Basic, `FailedBasicAuth`
  reports the realm only on failure.

## Caveats (from the example's own README)

- **At most one `Authorization` header.** Mixing `Authorization: Basic`
  and `Authorization: Bearer` is not supported by HTTP itself; the
  Bearer carrier should fall back to the `access_token` query/form
  field when Basic is also in play.
- **At most one scoped scheme per route.** If a spec declares two
  `oauth2` entries, both will see the same Bearer token — the runtime
  has no way to tell them apart at the wire level.
- **OpenAPI 2.0 only allows scopes on `oauth2`.** That's why the
  example uses `type: oauth2` for what is really plain JWT-with-claims.
- **All schemes share one principal type.** Aggregate intermediary
  state inside the principal struct itself.

## Run it end-to-end

The full runnable program — including the JWT keypair generator, a
curl exerciser script and the JWT-claims-based authorizers — lives
at
[`go-swagger/examples/composed-auth`](https://github.com/go-swagger/examples/tree/master/composed-auth).

The runtime side of that example is exactly what you see above; the
rest is application glue (DB lookups, JWT verification helpers, the
RSA keypair) that you'd write the same way against any HTTP framework.
