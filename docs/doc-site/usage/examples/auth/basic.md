---
title: HTTP Basic
weight: 20
description: |
  RFC 7617 Basic auth with realm and a WWW-Authenticate challenge on
  failure.
---

Same shape as the [API key](../api-key/) example, but with
username:password decoded by the runtime and a realm advertised on
the failure response.

## Spec

```yaml
securityDefinitions:
  basicAuth:
    type: basic

security:
  - basicAuth: []
```

## Wiring

{{< code file="auth/basic/main.go" lang="go" region="registerBasicAuth" >}}

`BasicAuthRealmCtx` is the context-aware variant of `BasicAuthRealm`;
the non-`*Ctx` form
[`security.BasicAuthRealm("petstore", fn)`](https://pkg.go.dev/github.com/go-openapi/runtime/security#BasicAuthRealm)
takes a `func(user, pass string) (any, error)` instead.

## Replying with `WWW-Authenticate` on 401

The runtime stashes the realm name in the request context when basic
auth has been attempted and failed. Recover it from a custom error
handler to render a proper challenge:

{{< code file="auth/basic/main.go" lang="go" region="failedBasicAuthChallenge" >}}

`FailedBasicAuth(r)` is the non-context spelling.

## Exercise

```sh
# Valid credentials
curl -i -u alice:s3cret http://127.0.0.1:8080/pets

# Missing or wrong credentials → 401 with challenge
curl -i http://127.0.0.1:8080/pets
# HTTP/1.1 401 Unauthorized
# WWW-Authenticate: Basic realm="petstore"
```

## When to combine with other schemes

Basic + Bearer is a common "either credential works" requirement.
That's the AND/OR composition case — see
[composed](../composed/) for how to declare and wire it.
