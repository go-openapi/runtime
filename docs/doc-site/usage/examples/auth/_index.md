---
title: Authentication & authorization
weight: 10
description: |
  Worked examples covering API keys, HTTP Basic, Bearer/JWT, OAuth2
  access-code flow, composed schemes, custom authorizers, and
  client-side credential attachment.
---

OpenAPI 2.0 defines four auth flavours; the runtime covers all four
plus the orthogonal `Authorizer` step. The pages below walk one
concrete scenario each — the first three cover the simplest cases,
the rest progressively layer on scopes, composition and custom
business rules.

## When to use which

| Situation                                                 | Start with                                                       |
|-----------------------------------------------------------|------------------------------------------------------------------|
| Single static API key in a header or query param          | [api-key](./api-key/)                                            |
| Username:password against a local store                   | [basic](./basic/)                                                |
| OAuth2 / OIDC bearer tokens with scope checks             | [bearer-jwt](./bearer-jwt/)                                      |
| You actually need the OAuth2 access-code dance with Google | [oauth2-access-code](./oauth2-access-code/)                      |
| Multiple schemes per operation (AND / OR composition)     | [composed](./composed/)                                          |
| RBAC / per-route business rules over the principal         | [custom-authorizer](./custom-authorizer/)                        |
| Client side — attaching credentials to outgoing requests   | [client-side](./client-side/)                                    |

For the conceptual model (interfaces, lifecycle, where each stage
fires), see [server / security](../../server/security/) and
[core / interfaces](../../core/interfaces/).

{{< children type="card" description="true" >}}
