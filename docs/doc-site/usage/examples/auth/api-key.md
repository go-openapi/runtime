---
title: API key (single scheme)
weight: 10
description: |
  Simplest server-side auth — a single API key carried in a header
  (or query parameter), validated against a static map.
---

The shortest path to a secured endpoint. Mirrors the
[`go-swagger/examples/authentication`](https://github.com/go-swagger/examples/tree/master/authentication)
example, in untyped form.

## Spec

```yaml
securityDefinitions:
  key:
    type: apiKey
    in: header
    name: X-Token

# default: every operation requires the key
security:
  - key: []
```

## Wiring

{{< code file="auth/apikey/main.go" lang="go" region="wireAPIKeyAuth" >}}

The scheme name passed to `RegisterAuth` (`"key"`) must match the
key under `securityDefinitions` in the spec.

## Exercise

```sh
# Valid token → 200
curl -i -H 'X-Token: abcdefuvwxyz' http://127.0.0.1:35307/customers/42

# Wrong / missing token → 401
curl -i -H 'X-Token: nope' http://127.0.0.1:35307/customers/42
# {"code":401,"message":"invalid api key"}
```

## Variations

- **Query param instead of header**: change `in:` to `query` in the
  spec and the second arg of `APIKeyAuth` to `"query"`. The token
  comes from `?api_key=…`.
- **Context-aware lookup** (DB call honouring request cancellation):
  use [`security.APIKeyAuthCtx`](../../../server/security/#why-ctx)
  instead — same idea, the callback gets the request `context.Context`.
- **Per-operation override**: a route can opt out by setting
  `security: []`; opt into a different scheme by replacing the list.
