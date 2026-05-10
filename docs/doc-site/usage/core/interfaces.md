---
title: Interfaces & layering
weight: 10
description: |
  The five core interfaces — Consumer, Producer, Authenticator,
  Authorizer, OperationHandler — and where each one fires on the
  client and server sides.
---

All interfaces live in the root package
[`github.com/go-openapi/runtime`](https://pkg.go.dev/github.com/go-openapi/runtime).
Each one comes with a companion `*Func` adapter so plain functions can be
used wherever an implementation is required.

## The five interfaces

### `Consumer` — bind a request body to a Go value

```go
type Consumer interface {
    Consume(io.Reader, any) error
}

type ConsumerFunc func(io.Reader, any) error
```

See the authoritative definition in
[godoc: `runtime.Consumer`](https://pkg.go.dev/github.com/go-openapi/runtime#Consumer).

Used on **both sides**:

- **server**: deserialize the inbound request body into the parameter struct
  matched to the operation
- **client**: deserialize the response body into the operation's typed
  result

### `Producer` — write a Go value to an HTTP response

```go
type Producer interface {
    Produce(io.Writer, any) error
}

type ProducerFunc func(io.Writer, any) error
```

See the authoritative definition in
[godoc: `runtime.Producer`](https://pkg.go.dev/github.com/go-openapi/runtime#Producer).

Used on **both sides**:

- **server**: serialize the operation handler's return value into the
  response body
- **client**: serialize a request body before sending

The split between `Consumer` and `Producer` is deliberate — request
deserialization and response serialization are independent concerns and a
given content type may want different behaviour on each side (think of
streaming uploads vs. buffered downloads).

### `Authenticator` — turn raw auth data into a principal

```go
type Authenticator interface {
    Authenticate(any) (bool, any, error)
}

type AuthenticatorFunc func(any) (bool, any, error)
```

See the authoritative definition in
[godoc: `runtime.Authenticator`](https://pkg.go.dev/github.com/go-openapi/runtime#Authenticator).

The three return values mean:

| Value     | Meaning                                                          |
|-----------|------------------------------------------------------------------|
| `bool`    | did this scheme apply to the request? (false ⇒ try the next one) |
| `any`     | the authenticated principal (whatever your app uses)             |
| `error`   | non-nil ⇒ scheme applied but failed                              |

Server-only. Built-in implementations for Basic, API key, Bearer and OAuth2
live in the [`security`](https://pkg.go.dev/github.com/go-openapi/runtime/security)
package, each with a context-aware `*Ctx` variant.

### `Authorizer` — gate the principal for this specific request

```go
type Authorizer interface {
    Authorize(*http.Request, any) error
}

type AuthorizerFunc func(*http.Request, any) error
```

See the authoritative definition in
[godoc: `runtime.Authorizer`](https://pkg.go.dev/github.com/go-openapi/runtime#Authorizer).

Authentication answers _who_; authorization answers _may they do this?_.
Authorizer runs after a principal has been resolved. A non-nil error blocks
the request.

Server-only. There is no built-in authorizer — you wire your own.

### `OperationHandler` — your business logic

```go
type OperationHandler interface {
    Handle(any) (any, error)
}

type OperationHandlerFunc func(any) (any, error)
```

See the authoritative definition in
[godoc: `runtime.OperationHandler`](https://pkg.go.dev/github.com/go-openapi/runtime#OperationHandler).

Server-only. The argument is the bound (and validated) parameter struct;
the return value is whatever `Producer` will then turn into the response
body. `error` propagates to the configured error handler.

## Server lifecycle — where each interface fires

For a request that reaches a matched route, the conventional pipeline
runs the following stages. Each stage is a separate `http.Handler` in
the chain, composable via `middleware.Builder`.

{{< mermaid align="center" zoom="true" >}}
flowchart TD
    req(((HTTP request)))
    router["Router<br/>match path/method against spec"]
    sec["Security · Context.Authorize<br/>Authenticator → principal<br/>Authorizer → may proceed?"]
    neg["ContentType / Accept negotiation<br/>pick Consumer + target Producer<br/>(part of BindValidRequest)"]
    bind["Binder<br/>path/query/header/body params<br/>— uses Consumer —"]
    val["Validator<br/>param validation + Validatable"]
    op["OperationExecutor<br/>call OperationHandler.Handle"]
    resp["Responder<br/>— uses Producer —"]
    out(((HTTP response)))

    req --> router --> sec --> neg --> bind --> val --> op --> resp --> out
{{< /mermaid >}}

A few things worth knowing:

- **The order above is a convention, not a runtime invariant.** It is what
  the runtime's untyped path
  ([`middleware.newRoutableUntypedAPI`](https://pkg.go.dev/github.com/go-openapi/runtime/middleware))
  does — it wraps the bind+validate closure with `newSecureAPI` so that
  security runs first — and what go-swagger's generated typed handlers do
  (each operation's `ServeHTTP` calls `Context.Authorize` *then*
  `Context.BindValidRequest` *then* the handler). You can compose a
  different chain via `middleware.Builder` if you have a reason to.
- **Security comes before binding and validation.** That way an
  unauthenticated request short-circuits with 401 without paying for
  parameter binding or body deserialization.
- **Auth is a single call site, not two.** `Context.Authorize` runs the
  configured authenticators in order and, on success, calls the optional
  `Authorizer`. An `Authenticator` returning `(false, nil, nil)` means
  "this scheme does not apply" and the next one is tried; a non-nil
  error short-circuits with 401.
- The pipeline's stages are documented in detail under
  [server / pipeline](../../server/pipeline/).

## Client lifecycle — where each interface fires

{{< mermaid align="center" zoom="true" >}}
flowchart TD
    gen["generated client method<br/>(GetPet, ListUsers, …)"]
    cop["ClientOperation<br/>{Params, Reader, …}<br/>(request descriptor)"]
    submit["Runtime.Submit"]
    enc["Producer<br/>encode body → *http.Request"]
    auth["AuthInfoWriter<br/>attach auth headers"]
    rt["Transport.RoundTrip<br/>(net/http)"]
    dec["Consumer<br/>decode response body"]
    res(((typed result)))

    gen --> cop --> submit
    submit --> enc --> rt
    submit --> auth --> rt
    rt --> dec --> res
{{< /mermaid >}}

`Authenticator` and `Authorizer` are **not used on the client**. Client-side
auth is attached through `AuthInfoWriter`, covered under
[client / authentication](../../client/auth/).

## Which interface goes where?

| Interface          | Server | Client | Notes                                                          |
|--------------------|:------:|:------:|----------------------------------------------------------------|
| `Consumer`         |   ✓    |   ✓    | request body in (server) / response body in (client)           |
| `Producer`         |   ✓    |   ✓    | response body out (server) / request body out (client)         |
| `Authenticator`    |   ✓    |        | scheme picks a principal from the request                      |
| `Authorizer`       |   ✓    |        | gates the principal for this request                           |
| `OperationHandler` |   ✓    |        | your business logic                                            |
| `Validatable` /<br>`ContextValidatable` | ✓ | ✓ | model self-validation; details in [validation](../validation/) |
