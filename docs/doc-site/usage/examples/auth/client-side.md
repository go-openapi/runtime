---
title: Client-side credentials
weight: 70
description: |
  Attaching auth information to outgoing requests — Basic, API key,
  Bearer, composed writers, and a custom HMAC signer.
---

Server-side authentication is the *Authenticator* story. Client-side
authentication is the *ClientAuthInfoWriter* story — pure encoding:
take credentials, set the right header / query parameter on the
outgoing request. See [client / authentication](../../../client/auth/)
for the full reference; this page is a recipe collection.

## Built-in writers

{{< code file="auth/clientside/main.go" lang="go" region="builtinWriters" >}}

`DefaultAuthentication` is used for any operation that does not
specify its own. Per-operation override:

{{< code file="auth/clientside/main.go" lang="go" region="perOperationOverride" >}}

## Composing multiple credentials

For APIs that require more than one credential header on the same
request (an API key plus a bearer token, say):

{{< code file="auth/clientside/main.go" lang="go" region="composeWriters" >}}

`Compose` skips nil writers; the first one to return an error
short-circuits the chain.

## Refreshing OAuth2 tokens

[`BearerToken`](https://pkg.go.dev/github.com/go-openapi/runtime/client#BearerToken)
captures a fixed string. For tokens that need to refresh mid-session,
wrap a [`golang.org/x/oauth2.TokenSource`](https://pkg.go.dev/golang.org/x/oauth2#TokenSource):

```go
// requires `golang.org/x/oauth2` — left inline because the doc-examples
// module intentionally avoids pulling in that dependency.
import (
    "github.com/go-openapi/runtime"
    "github.com/go-openapi/strfmt"
    "golang.org/x/oauth2"
)

func OAuth2(src oauth2.TokenSource) runtime.ClientAuthInfoWriter {
    return runtime.ClientAuthInfoWriterFunc(func(r runtime.ClientRequest, _ strfmt.Registry) error {
        tok, err := src.Token() // refreshes when expired
        if err != nil {
            return err
        }
        return r.SetHeaderParam("Authorization", "Bearer "+tok.AccessToken)
    })
}

rt.DefaultAuthentication = OAuth2(myTokenSource)
```

## Custom: HMAC body signing

Sign the body with a shared secret and attach the signature as a
header:

{{< code file="auth/clientside/main.go" lang="go" region="hmacSignatureWriter" >}}

Then wire it on the runtime:

```go
rt.DefaultAuthentication = HMACSignature("k1", sharedSecret)
```

The runtime calls `AuthenticateRequest` after the operation's
parameters have been bound — so for buffered bodies `r.GetBody()`
returns the encoded payload. For streaming bodies (multipart, raw
streams) the runtime arranges a body-copy closure so the signer sees
the bytes that go on the wire; see
[client / requests](../../../client/requests/#what-happens-during-a-submitcontext-call)
for the exact assembly path.

## Explicit "no auth"

For operations whose spec lists a security requirement that should be
satisfied by sending nothing (rare but legal):

{{< code file="auth/clientside/main.go" lang="go" region="passThroughAuth" >}}

A nil writer would have the same effect — [`PassThroughAuth`](https://pkg.go.dev/github.com/go-openapi/runtime/client#PassThroughAuth)
is the explicit version, useful when you want the intent to read
clearly in review.
