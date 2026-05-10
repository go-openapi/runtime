---
title: Keep-alive in the runtime client
weight: 30
description: |
  How `go-openapi/runtime` reuses TCP connections, what the kernel
  and the HTTP transport actually do for you, and where it goes
  wrong when there is a NAT gateway, proxy, or firewall between
  your client and the server.
---

How `go-openapi/runtime` reuses TCP connections, what the kernel and the
HTTP transport actually do for you, and where it goes wrong when there
is a NAT gateway, proxy, or firewall between your client and the server.

Concrete: the reference for "I get `context deadline exceeded` after a
quiet period" — issue #336 is the canonical example.

> Scope: client-side `Runtime`. Server-side keep-alive (`http.Server`'s
> own timers) is summarised briefly at the end, with pointers into the
> Go stdlib docs.

## TL;DR

If your client lives behind a NAT gateway, a load balancer, or a firewall
with an idle conntrack timeout (AWS NAT: **350 seconds**; many corporate
firewalls: a few minutes), and you see `context deadline exceeded` on
requests that follow a quiet period:

1. **Check what your `Runtime.Transport` is.** If you let `client.New`
   pick the default (`http.DefaultTransport`), you already get
   `IdleConnTimeout = 90s` and `Dialer.KeepAlive = 30s`. Those defeat
   most NAT timeouts.
2. **If you replaced the Transport** (for TLS config, a proxy, etc.),
   you almost certainly lost those defaults. Reinstate them.
3. **On Go 1.23+, set an explicit
   [`net.Dialer.KeepAliveConfig`](https://pkg.go.dev/net#Dialer)** —
   the bare `KeepAlive` field only sets the probe *interval*, not the
   *idle delay before probing starts*. On Linux the kernel default for
   the idle delay is often **7200 seconds** (two hours), so probes
   never fire before a 350s NAT timeout drops your conntrack.
4. **Do not** reach for
   [`Runtime.EnableConnectionReuse`](https://pkg.go.dev/github.com/go-openapi/runtime/client#Runtime.EnableConnectionReuse).
   The name is misleading — it does not control TCP keepalive or NAT
   timeouts. See "the misnomer" below.

A recipe at the bottom of this document covers the cloud / NAT case.

## Two distinct things named "keep-alive"

The word "keep-alive" is used for two unrelated mechanisms operating at
different layers. The runtime, the stdlib, and the OS all speak about
"keep-alive" without always disambiguating, which is the root of most
confusion.

### HTTP keep-alive (application layer)

`Connection: keep-alive` is an HTTP/1.1 default. It means **the same TCP
connection serves multiple HTTP request/response pairs**. The client
sends request 1, reads response 1, sends request 2 *on the same socket*,
reads response 2, and so on, until either side closes.

In Go, `http.Transport` keeps a pool of idle connections per host. After
a response body is fully read and closed, the connection goes back to
the pool. The next request to the same host may pick a connection from
the pool instead of dialling a new one. Skipping the dial saves a TCP
handshake plus, for HTTPS, a TLS handshake — typically tens to hundreds
of milliseconds per request.

### TCP keepalive (kernel / socket layer)

`SO_KEEPALIVE` is a socket option asking the kernel to **send periodic
empty ACK packets** on an otherwise-idle TCP connection. The peer
acknowledges them. Two consequences:

1. **Dead-peer detection.** If the peer disappears (machine rebooted,
   network partitioned), the kernel sees the missing ACKs and tears
   the connection down. Without keepalive, a half-open connection can
   linger indefinitely.
2. **Conntrack / NAT keep-alive.** A NAT gateway or stateful firewall
   maintains a *connection tracking* (conntrack) entry per TCP flow
   passing through it. The entry is dropped after some idle period —
   AWS NAT uses 350 seconds, many enterprise firewalls use 60s–15min.
   Once the entry is dropped, packets arriving for that flow are
   either silently discarded or rejected with a RST that may not
   reach the original sender. **Periodic TCP keepalive packets count
   as live traffic**, so the NAT keeps the entry fresh.

The first three of the four mechanisms below are HTTP keep-alive
concerns; the fourth is the kernel/TCP one.

| Knob | Layer | Default | What it controls |
|---|---|---|---|
| `http.Transport.DisableKeepAlives` | HTTP | `false` (keep-alive on) | Whether a TCP conn serves more than one HTTP request |
| `http.Transport.MaxIdleConns` / `MaxIdleConnsPerHost` | HTTP | 100 / 2 | Idle-pool sizing |
| `http.Transport.IdleConnTimeout` | HTTP | 90s | How long an idle conn stays in the pool before close |
| `net.Dialer.KeepAlive` (+ `KeepAliveConfig` on Go 1.23+) | TCP | 30s | Whether and how the kernel sends keepalive probes on dialled connections |

If you only remember one thing: **HTTP-layer settings decide whether the
runtime *reuses* a connection. TCP-layer settings decide whether a
through-the-network proxy *still believes the connection exists*.** A
mismatch produces issue #336.

## What Go does for you, by default

`http.DefaultTransport` is the transport
[`client.New`](https://pkg.go.dev/github.com/go-openapi/runtime/client#New)
sets on every fresh `Runtime`. Its defaults, as of recent Go:

```go
// from net/http
DefaultTransport = &http.Transport{
    Proxy:                 http.ProxyFromEnvironment,
    DialContext: (&net.Dialer{
        Timeout:   30 * time.Second,
        KeepAlive: 30 * time.Second, // TCP keepalive interval
    }).DialContext,
    ForceAttemptHTTP2:     true,
    MaxIdleConns:          100,
    IdleConnTimeout:       90 * time.Second,
    TLSHandshakeTimeout:   10 * time.Second,
    ExpectContinueTimeout: 1 * time.Second,
}
```

Read the way most cloud-deployed Go services need it:

- `IdleConnTimeout = 90s` is less than the AWS NAT 350s timeout, so an
  idle pooled connection is closed by Go before NAT drops it.
- `Dialer.KeepAlive = 30s` enables TCP keepalive probes every 30s, so
  *active* connections survive long NAT timeouts even when the
  application isn't sending data.

For typical cases, these defaults are correct. **You only need to think
about this if you replaced the Transport, or if your environment has an
unusual idle timeout.**

## How the runtime wires this

`Runtime.Transport` is the `http.RoundTripper` used for every outbound
request. Three things to know:

1. The default is `http.DefaultTransport`, with the values above.
2. Replacing `rt.Transport = ...` with a custom transport
   **completely overrides** the defaults — you inherit nothing unless
   you copy what `http.DefaultTransport` sets.
3. `Runtime.SetDebug(true)` does not affect keep-alive at all — it
   only logs requests/responses.

### The misnomer — `Runtime.EnableConnectionReuse`

`Runtime.EnableConnectionReuse()` is the method most users find when
searching for "keep-alive" or "connection reuse" in this codebase. The
name suggests it controls whether connections are pooled and reused. It
does not.

What it actually does: wraps `Runtime.Transport` in a `RoundTripper`
that, after every response, **drains any unread bytes from the response
body** before `Close`. The reason: Go's `http.Transport` will only
return a connection to the idle pool if the response body was fully
read. If your handler stops reading early — for example, you only need
the HTTP status and skip the body — the connection is not reusable, and
the next request will pay the cost of a new dial + handshake.

So `EnableConnectionReuse` is a narrow fix for one specific pattern: code
that doesn't fully read response bodies. It has **no effect on**:

- TCP keepalive packets;
- whether the connection survives a NAT idle timeout;
- the size of the idle pool;
- the idle timeout in the pool;
- any other connection-lifecycle concern.

If you ended up here following the issue #336 trail: this method will
not help you. A future runtime release will either rename this method
to something narrow and honest, or fold the body-draining behaviour
into a default-on path so users no longer have to know about it.

## The NAT idle-timeout failure mode

This is the scenario in issue #336. Walk through it once and the symptom
becomes recognisable:

1. The client makes a request. Go dials a fresh TCP connection through
   the NAT gateway. NAT creates a conntrack entry. Request completes;
   the connection goes into Go's idle pool.
2. The application is quiet for **more than 350 seconds**.
3. Go's idle pool has not yet evicted the connection (if you increased
   `IdleConnTimeout` past 350s, or if you have a custom transport that
   doesn't set it). Or the conn is "active" because something is
   waiting on it, just not sending data — long polling, server-sent
   events, slow streaming response.
4. **NAT drops the conntrack entry.** No notification to either side.
5. The application makes its next request. Go picks the still-pooled
   connection. The TCP stack believes it is fine; it sends.
6. **Packets disappear at the NAT.** The server never sees the request,
   the client never sees a response. From the application's view, the
   request hangs.
7. Eventually the request's context deadline fires:
   `context deadline exceeded`.

The same shape applies to any stateful network appliance between you and
the server: load balancers, corporate firewalls, IPSec tunnels.

## Solutions

### Rely on the defaults (preferred)

If you can: use `http.DefaultTransport`, do not replace
`rt.Transport`. `IdleConnTimeout=90s` and `Dialer.KeepAlive=30s`
together cover the common NAT and firewall idle timeouts. No further
configuration needed.

### Custom Transport — reinstate the defaults

When you build a custom transport (for `TLSClientConfig`, an HTTP proxy
URL, a `MaxIdleConnsPerHost` change, etc.), **start from the
`http.DefaultTransport` values, then override only what you need**:

```go
import (
    "net"
    "net/http"
    "time"
)

rt.Transport = &http.Transport{
    Proxy: http.ProxyFromEnvironment,
    DialContext: (&net.Dialer{
        Timeout:   30 * time.Second,
        KeepAlive: 30 * time.Second,
    }).DialContext,
    ForceAttemptHTTP2:     true,
    MaxIdleConns:          100,
    IdleConnTimeout:       90 * time.Second,
    TLSHandshakeTimeout:   10 * time.Second,
    ExpectContinueTimeout: 1 * time.Second,

    TLSClientConfig: yourTLSConfig, // <- your actual override
}
```

The single most common bug is omitting the `Dialer`. A literal of the
form `&http.Transport{TLSClientConfig: ...}` with no `DialContext`
uses Go's net default dialler, which has **no keepalive at all**.

### Explicit `KeepAliveConfig` (Go 1.23+)

The bare `net.Dialer.KeepAlive` field sets the probe interval. On Linux,
the kernel does not start sending probes until a separate idle delay
elapses, and that idle delay defaults to **7200 seconds** at the
`tcp_keepalive_time` sysctl. With AWS NAT's 350s timeout, the probes
never start in time.

Go 1.23 introduced `net.Dialer.KeepAliveConfig`, which lets you set the
idle delay explicitly so the kernel does not depend on `tcp_keepalive_time`:

```go
DialContext: (&net.Dialer{
    Timeout: 30 * time.Second,
    KeepAliveConfig: net.KeepAliveConfig{
        Enable:   true,
        Idle:     60 * time.Second,  // wait 60s of idleness, then start probing
        Interval: 30 * time.Second,  // send a probe every 30s
        Count:    4,                 // drop the conn after 4 missed probes
    },
}).DialContext,
```

With these numbers, after 60 seconds of silence the kernel starts
sending probes, well before the 350s NAT timeout — the conntrack stays
fresh, the application sees no surprises.

### Other levers

- `http.Transport.IdleConnTimeout` set to less than the NAT timeout
  forces Go to close idle connections before NAT can drop them. The
  next request then dials fresh.
- `http.Transport.DisableKeepAlives = true` opts out of HTTP keep-alive
  entirely — every request gets a fresh TCP connection. Simple and
  correct, but trades a handshake cost on every request. Reasonable
  for low-volume clients; pathological for high-volume ones.

## Diagnosing keep-alive problems

When you suspect a keep-alive issue:

- **Confirm the symptom shape.** "Context deadline exceeded" after a
  quiet period is the fingerprint of a dropped conntrack. If the
  failures happen *under load*, it's almost certainly something else.
- **Check the Transport.** Print or log `rt.Transport` early in your
  application; if it is `*http.Transport`, inspect `IdleConnTimeout`
  and the dialler's `KeepAlive` / `KeepAliveConfig`. Many subtle bugs
  vanish at this step.
- **Use `httptrace`.** The stdlib's
  [`net/http/httptrace`](https://pkg.go.dev/net/http/httptrace) package
  surfaces the connection lifecycle — `GotConn`, `PutIdleConn`,
  `ConnectStart`, `TLSHandshakeStart`, etc. When you see `GotConn`
  with `Reused: true` immediately followed by a hang, you have caught
  a stale pooled connection. (Future runtime versions may surface
  this via a built-in helper; see the roadmap.)
- **On Linux, inspect kernel state.** `ss -t -o` shows the keepalive
  timer for each active socket; `cat /proc/sys/net/ipv4/tcp_keepalive_*`
  shows the kernel defaults; `conntrack -L` (where available) shows
  the NAT side.
- **`tcpdump` on the client.** Look for outbound packets with no
  inbound response after the symptom appears. Confirms the NAT-drop
  hypothesis.

## Server-side, briefly

A server's keep-alive behaviour is governed by
[`http.Server`](https://pkg.go.dev/net/http#Server), not by anything in
the runtime middleware:

- `Server.IdleTimeout` — how long a kept-alive connection waits for the
  next request before the server closes it.
- `Server.ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout` — bound the
  time spent on individual phases; expiry closes the connection.

The runtime's server middleware does not override these. If your server
sits behind a NAT or load balancer with an idle timeout, set
`Server.IdleTimeout` to a value below that timeout so the server
proactively closes idle connections — clients on Go will simply dial
again on their next request without surfacing an error.

## Recipe — `Runtime` for cloud / NAT environments

The construction below is the conservative starting point for a client
deployed in AWS, GCP, or behind any stateful network appliance with an
idle timeout. Adjust the timing constants if you have measurements; do
not adjust them on intuition alone.

```go
package main

import (
    "net"
    "net/http"
    "time"

    "github.com/go-openapi/runtime/client"
)

func newClient(host, basePath string) *client.Runtime {
    rt := client.New(host, basePath, []string{"https"})

    rt.Transport = &http.Transport{
        Proxy: http.ProxyFromEnvironment,
        DialContext: (&net.Dialer{
            Timeout: 30 * time.Second,
            // Go 1.23+: explicit idle delay; bare KeepAlive=30s is
            // not enough on Linux because the kernel idle default
            // (tcp_keepalive_time) is often 7200s.
            KeepAliveConfig: net.KeepAliveConfig{
                Enable:   true,
                Idle:     60 * time.Second,
                Interval: 30 * time.Second,
                Count:    4,
            },
        }).DialContext,
        ForceAttemptHTTP2:     true,
        MaxIdleConns:          100,
        IdleConnTimeout:       60 * time.Second, // < AWS NAT's 350s
        TLSHandshakeTimeout:   10 * time.Second,
        ExpectContinueTimeout: 1 * time.Second,
    }

    return rt
}
```

If you cannot move to Go 1.23, fall back to:

```go
DialContext: (&net.Dialer{
    Timeout:   30 * time.Second,
    KeepAlive: 30 * time.Second,
}).DialContext,
IdleConnTimeout: 60 * time.Second,
```

and rely on the `IdleConnTimeout` to evict pooled connections before
NAT does. The kernel keepalive probes may or may not fire in time
depending on `tcp_keepalive_time`, but at least your idle pool is
self-policing.

## Reference

- [`net/http.Transport`](https://pkg.go.dev/net/http#Transport)
- [`net.Dialer`](https://pkg.go.dev/net#Dialer),
  [`net.KeepAliveConfig`](https://pkg.go.dev/net#KeepAliveConfig) (Go 1.23+)
- [`net/http/httptrace.ClientTrace`](https://pkg.go.dev/net/http/httptrace#ClientTrace)
- Client transport: `client/runtime.go` (`Runtime.Transport`, `Runtime.New`)
- The misnomer: `client/keepalive.go`
  ([`KeepAliveTransport`](https://pkg.go.dev/github.com/go-openapi/runtime/client#KeepAliveTransport),
  [`Runtime.EnableConnectionReuse`](https://pkg.go.dev/github.com/go-openapi/runtime/client#Runtime.EnableConnectionReuse))
- Issue #336: <https://github.com/go-openapi/runtime/issues/336>
