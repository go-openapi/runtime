---
title: OAuth2 access-code (Google)
weight: 50
description: |
  Full OAuth2 access-code handshake against Google — login redirect,
  callback handler, token exchange and protected operations.
---

Mirrors
[`go-swagger/examples/oauth2`](https://github.com/go-swagger/examples/tree/master/oauth2).
Most of this example is OAuth2-flow plumbing (redirect, callback,
token exchange) that lives in *your* code, not in the runtime — the
runtime only enters the picture for the protected endpoints, where
the bearer token is validated.

The [bearer-jwt](../bearer-jwt/) example is the right starting point
if all you need is *validating* an inbound bearer; come here when you
also want to *issue* the redirect dance.

## Spec

```yaml
securityDefinitions:
  OauthSecurity:
    type: oauth2
    flow: accessCode
    authorizationUrl: 'https://accounts.google.com/o/oauth2/v2/auth'
    tokenUrl:         'https://www.googleapis.com/oauth2/v4/token'
    scopes:
      user:  regular user
      admin: administrative

security:
  - OauthSecurity: [user]

paths:
  /login:
    get:
      security: []          # public — kicks off the redirect
  /auth/callback:
    get:
      security: []          # public — receives the code from Google
  /customers:
    get:
      # uses the default `OauthSecurity: [user]`
      ...
```

## Application configuration

You'll need a registered OAuth2 client at
<https://console.cloud.google.com/apis/credentials/> and an exact-match
callback URL.

```go
import (
    oidc "github.com/coreos/go-oidc"
    "golang.org/x/oauth2"
)

var (
    state        = "foobar"                            // single-shot CSRF token; see note below
    clientID     = "<your-client-id>"
    clientSecret = "<your-client-secret>"
    callbackURL  = "http://127.0.0.1:12345/api/auth/callback"
    userInfoURL  = "https://www.googleapis.com/oauth2/v3/userinfo"

    config = oauth2.Config{
        ClientID:     clientID,
        ClientSecret: clientSecret,
        Endpoint: oauth2.Endpoint{
            AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
            TokenURL: "https://www.googleapis.com/oauth2/v4/token",
        },
        RedirectURL: callbackURL,
        Scopes:      []string{oidc.ScopeOpenID, "profile", "email"},
    }
)
```

## Wiring

{{< code file="auth/oauth2/main.go" lang="go" region="wireOauth2AccessCode" >}}

`validateAtUserInfoURL` is a plain HTTP call to Google's userinfo
endpoint with the bearer token — see the
[full implementation](https://github.com/go-swagger/examples/blob/master/oauth2/restapi/implementation.go)
in the sibling repo.

> **State parameter, briefly**: the example uses a global string for
> brevity. In production this MUST be a per-session unguessable
> value, stored alongside the user's session and validated on the
> callback — otherwise CSRF on the redirect.

## Exercise

```sh
# 1. Visit the login URL in a browser
open http://127.0.0.1:12345/api/login
# → redirected to Google sign-in
# → after consent, redirected back to /auth/callback
# → the response includes the access_token

# 2. Call a protected endpoint with that token
curl -i -H "Authorization: Bearer $TOKEN" http://127.0.0.1:12345/api/customers

# Wrong token → 401
curl -i -H "Authorization: Bearer garbage" http://127.0.0.1:12345/api/customers
# {"code":401,"message":"unauthenticated for invalid credentials"}
```

## Run the full example

The complete runnable program — including the userinfo validator,
the redirect/callback handlers wired through middleware, and the
client-secret bootstrap — lives at
[`go-swagger/examples/oauth2`](https://github.com/go-swagger/examples/tree/master/oauth2).
Clone it, drop your Google client ID/secret into
`restapi/implementation.go`, and run.
