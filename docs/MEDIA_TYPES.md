# Media-type selection

How `go-openapi/runtime` parses, matches, and negotiates HTTP media types,
on both the server and client sides. The reference for the rules behind a
415, a 406, or a 400 you see in production.

> Scope: `Content-Type` and `Accept` headers, both inbound and outbound.
> `Accept-Encoding` is mentioned briefly. Charset, language, and version
> tags are treated as opaque parameters under the rules below.

## At a glance — error mapping

| Outcome | HTTP | Where it's raised |
|---|---|---|
| Inbound `Content-Type` does not parse | **400** Bad Request | [`runtime.ContentType`](https://pkg.go.dev/github.com/go-openapi/runtime#ContentType), [`errors.ParseError`](https://pkg.go.dev/github.com/go-openapi/errors#ParseError) |
| Inbound `Content-Type` is well-formed but not in the operation's `consumes` | **415** Unsupported Media Type | [`errors.InvalidContentType`](https://pkg.go.dev/github.com/go-openapi/errors#InvalidContentType) |
| `Accept` cannot be satisfied by the operation's `produces` | **406** Not Acceptable | [`errors.InvalidResponseFormat`](https://pkg.go.dev/github.com/go-openapi/errors#InvalidResponseFormat) |
| No consumer registered for an otherwise-allowed `Content-Type` | **500** Internal Server Error | server-side configuration error |

## The shared model — `mediatype.MediaType`

Both sides use the same parser and value type:

```go
import "github.com/go-openapi/runtime/server-middleware/mediatype"

mt, err := mediatype.Parse("application/json;charset=utf-8;q=0.8")
// mt.Type    = "application"
// mt.Subtype = "json"
// mt.Params  = {"charset": "utf-8"}      // parameter keys lowercased
// mt.Q       = 0.8                       // q is extracted, not stored in Params
```

### Casing

- `Type`, `Subtype`, parameter keys → lowercased on parse.
- Parameter values → preserved verbatim.
- Comparisons of parameter values are **case-insensitive**
  (`charset=UTF-8` matches `charset=utf-8`, the convention for charset, version, etc.).

### Wildcards

`*/*` and `type/*` are accepted on either side of a comparison.
`*/subtype` is invalid per RFC 7231 §5.3.2 and `Parse` rejects it.

### Malformed input

Every `Parse` failure wraps the sentinel `mediatype.ErrMalformed`,
so callers can distinguish "client sent garbage" from "client sent
something well-formed that nothing here accepts":

```go
_, err := mediatype.Parse(headerValue)
if errors.Is(err, mediatype.ErrMalformed) {
    // 400 Bad Request territory
}
```

## The matching rule

`MediaType.Matches(other)` is **asymmetric**. The receiver is the *bound*
(an allowed entry on the server side, or a candidate offer when matching
against an `Accept` entry); the argument is the *constraint* (the actual
incoming request, or the `Accept` entry being satisfied).

The rule:

1. Bare `type/subtype` must agree (with wildcards on either side).
2. If the receiver carries **no parameters**, any constraint is accepted
   regardless of its parameters.
3. Otherwise every `(key, value)` pair on the constraint must be present
   on the receiver, with case-insensitive value comparison. The receiver
   may carry **additional** parameters that the constraint does not list.

q-values are **not** considered by `Matches` — they are the negotiator's
concern, handled inside `Set.BestMatch`.

The same direction is used in both call sites:

| Call | Bound (receiver) | Constraint (argument) |
|---|---|---|
| Inbound validation | each entry in `consumes` | the request's `Content-Type` |
| `Accept` negotiation | each candidate offer | each `Accept` entry |

The asymmetry is intrinsic to the semantics ("loose if the bound has no
params, otherwise the constraint must be a subset"), not to which side is
the server.

## Server side — inbound `Content-Type` validation

Flow when a request arrives with a body:

```
runtime.HasBody(r)               ── early-out for bodyless requests
  ↓
runtime.ContentType(r.Header)    ── 400 here if the header is malformed
  ↓
validateContentType(consumes, ct)
  ├─ malformed actual            → 400 errors.ParseError      (defensive)
  ├─ no entry matches            → 415 errors.InvalidContentType
  └─ match                       → continue to consumer dispatch
  ↓
route.Consumers[ct]              ── 500 if no codec registered
```

`validateContentType` is a thin wrapper around
[`mediatype.MatchFirst`](https://pkg.go.dev/github.com/go-openapi/runtime/server-middleware/mediatype#MatchFirst).
It short-circuits on the first allowed entry that accepts the actual —
not the most specific match. For ranked matching use `Set.BestMatch`.

### What "missing `Content-Type`" does

When the request body is non-empty but the header is missing,
`runtime.ContentType` substitutes the package-level default
(`runtime.DefaultMime` = `application/octet-stream`). The validator
then matches that default against the operation's `consumes`. So a
request with a body and no `Content-Type` typically yields **415**
unless the operation lists `application/octet-stream`.

### Parameter honouring (since v0.30)

Before v0.30, parameters were stripped on both sides before matching:
`Content-Type: text/plain;charset=ascii` would pass against
`consumes: [text/plain;charset=utf-8]`. Since v0.30 this is rejected
(charset values disagree). The fix landed with PR #426 (issue #136).

## Server side — outbound `Accept` negotiation

[`negotiate.ContentType(r, offers, defaultOffer, opts...)`](https://pkg.go.dev/github.com/go-openapi/runtime/server-middleware/negotiate#ContentType)
reads the request's `Accept` header(s), parses each entry,
ranks the offers, and returns the winning offer (a string from the
`offers` slice). If nothing matches, `defaultOffer` is returned.

### Ranking

Per RFC 7231 §5.3.2, in order:

1. Highest **q-value** (`q=0` excludes an offer entirely).
2. Highest **specificity** of the matched `Accept` entry
   (`type/subtype;params` > `type/subtype` > `type/*` > `*/*`).
3. Earliest position in the `offers` slice.

### Multiple `Accept` headers

Per RFC 7230 §3.2.2, multiple `Accept` headers are equivalent to a single
comma-joined value. The negotiator joins before parsing, so all entries
contribute to the decision regardless of how the client batched them.

### Parameter honouring and the opt-out

Same v0.30 change as inbound validation. An `Accept` entry of
`text/plain;charset=utf-8` matches an offer of bare `text/plain` (offer
carries no constraint), but **not** `text/plain;charset=ascii`.

To restore the looser pre-v0.30 behaviour for one operation:

```go
chosen := negotiate.ContentType(r, offers, "",
    negotiate.WithIgnoreParameters(true),
)
```

…or server-wide, threaded through the middleware `Context`:

```go
ctx := middleware.NewContext(spec, api, nil).SetIgnoreParameters(true)
```

The opt-out exists for applications whose producers and `Accept` clients
use mismatched charset or version params that they treat as
informational.

### Codec dispatch is keyed by bare type

The negotiator returns the verbatim offer (parameters preserved) and the
runtime sets `Content-Type` from it. Codec dispatch is a separate step:
the runtime looks up the producer in `route.Producers`, which is a
`map[string]Producer` keyed by the **bare** `type/subtype` (no params).
You will see calls to `normalizeOffer(format)` and
`normalizeOffers(...)` in the middleware and the router doing exactly
this stripping — they are about map lookup, not about negotiation.

The practical consequence: you cannot register two different producers
for the same bare type that differ only by parameters
(`text/plain;charset=utf-8` vs `text/plain;charset=ascii`). They would
collide on the bare-type key. The negotiator can still **choose**
between two such offers (parameters are honoured during matching), but
the codec invoked is the single one registered under the bare key.

If you need parameter-specific encoding, do it inside one producer and
inspect the negotiated `Content-Type` from the response writer.

## Client side — outbound `Content-Type`

Selection runs in two stages. Stage 1 picks a candidate from the
operation's `consumes` list before the payload is known; Stage 2 runs
inside `buildHTTP` after the request writer has populated the payload,
and may upgrade Stage 1's choice when the payload is a stream.

### Stage 1 — `pickConsumesMediaType`

Source: `client/runtime.go`.

```go
cmt := pickConsumesMediaType(operation.ConsumesMediaTypes, r.Producers, r.DefaultMediaType)
```

1. If `multipart/form-data` is one of the entries, prefer it (it streams
   and preserves per-file `Content-Type`). Resolves issue #286.
2. Otherwise the first non-empty entry that is **either** a structural
   mime (`multipart/form-data`, `application/x-www-form-urlencoded`)
   **or** has a producer registered in `Runtime.Producers`. This skips
   spec entries the client cannot serialise — useful when the spec
   lists a vendor mime first and a registered alternative second.
   Closes part of issues #32 and #386.
3. If nothing in the list is registered, the first non-empty entry is
   returned anyway so the gate at the call site emits its
   `none of producers: …` diagnostic.
4. Falls back to `Runtime.DefaultMediaType` (`application/json` by
   default) only when the list is empty (or all empty strings).

Stage 1 cannot see the payload — the request writer hasn't run yet —
so its choice is "best effort given only the spec and the registered
producers."

### Stage 2 — `setStreamContentType`

Source: `client/request.go`. Runs inside `buildHTTP` after the writer
has populated `r.payload`. For stream payloads (`io.Reader`,
`io.ReadCloser`) only — the producer is bypassed in this branch, so
the wire header is the only place where the body's actual MIME type
is asserted.

Three checks, in priority order:

1. **Explicit `SetHeaderParam("Content-Type", …)`.** The historical
   header escape hatch wins over every derivation. If the writer set
   `Content-Type` during `WriteToRequest`, the runtime keeps it as-is.
   This was not the original purpose of `SetHeaderParam`, but it has
   become the natural way to say "send THIS exact header", and we
   honour it. Caveat: the user is then responsible for matching their
   declared header to their actual body bytes.

2. **Payload-declared content type.** If `r.payload` implements the
   exported [`runtime.ContentTyper`](https://pkg.go.dev/github.com/go-openapi/runtime#ContentTyper)
   interface and returns a non-empty value, that value wins. The
   value declares its own nature — useful for line-delimited formats,
   custom MIME types, or any case where the spec offers no matching
   entry. The same interface is also consulted on each part of a
   multipart file upload.

3. **Octet-stream upgrade.** When neither of the above applies, and
   `application/octet-stream` is in the operation's `consumes` list
   AND a producer is registered for it, the wire header is upgraded
   from the picker's choice to octet-stream — a safer "raw bytes"
   claim than a structural mime like JSON.

If none of the three checks fire, the picker's `mediaType` from
Stage 1 is used as the terminal fallback.

#### Non-stream paths are deliberately not honoured

`SetHeaderParam("Content-Type", …)` and `runtime.ContentTyper` are
honoured **only** for stream payloads. Non-stream paths have
structural constraints that conflict with arbitrary user-supplied
content types:

- **`struct` / `[]byte` payloads** — the producer is dispatched off
  `mediaType`. Honouring an arbitrary user header here would mean
  either swapping the producer (complex) or sending a body that
  doesn't match the declared header (still a lie).
- **Multipart bodies** — the runtime owns the `Content-Type` header
  because of the boundary parameter requirement.
- **URL-encoded forms** — the body is form-encoded; lying about the
  type would break parsing on the server.

Users with these payload shapes who need a custom content type
should adjust the operation's `consumes` list (so the picker selects
the right entry) or register a producer under the desired MIME.

### Wire `Content-Type` matrix

| Payload | `SetHeader Content-Type` | declares `ContentType()` | octet-stream offered + registered | Wire `Content-Type` |
|---|---|---|---|---|
| stream | set | — | — | the SetHeader value |
| stream | unset | yes, non-empty | — | declared value |
| stream | unset | no / empty | yes | `application/octet-stream` |
| stream | unset | no / empty | no | picker's choice (best-effort; may misrepresent body) |
| `struct` | (ignored) | — | — | picker's choice (producer runs) |
| `[]byte` | (ignored) | — | — | picker's choice (producer runs; e.g. JSON producer base64-encodes) |

### Declaring a stream's MIME type

Wrap the reader in a type that satisfies [`runtime.ContentTyper`](https://pkg.go.dev/github.com/go-openapi/runtime#ContentTyper):

```go
type ndjsonStream struct { io.Reader }

func (n *ndjsonStream) ContentType() string { return "application/x-ndjson" }

// in your params writer:
return r.SetBodyParam(&ndjsonStream{Reader: myReader})
```

The wire `Content-Type` will be `application/x-ndjson` regardless of
which entry the picker chose from the operation's `consumes`.

### Codec registration

The client transport ships with a fixed codec set (JSON, YAML, XML, CSV,
text, HTML, byte-stream). Register additional MIME types directly:

```go
rt := client.New(host, basePath, schemes)
rt.Consumers["application/problem+json"] = runtime.JSONConsumer()
rt.Producers["application/problem+json"] = runtime.JSONProducer()
```

See [FAQ § custom MIME types](FAQ.md#how-do-i-register-custom-mime-types-eg-applicationproblemjson).

### Known gaps

- **Issue [#385](https://github.com/go-openapi/runtime/issues/385) /
  [#33](https://github.com/go-openapi/runtime/issues/33)** — The codec
  set is hardcoded; it is not derived from the spec. Apps that don't
  declare an exotic `consumes`/`produces` carry codecs they will never
  use. Tracked as Track A.2 in the modularization roadmap.
- **`[]byte` payloads.** A `[]byte` flows through the picker's chosen
  producer. The JSON producer base64-encodes it as a JSON string. If
  you want raw bytes on the wire, wrap as `bytes.NewReader([]byte{…})`
  — it then takes the stream path and the Stage-2 octet-stream
  upgrade applies.

### What changed in v0.30 (client-side outbound)

Four behaviour deltas vs. v0.29. Three are confined to **stream
payloads** (`io.Reader`, `io.ReadCloser`); the fourth touches the
Stage-1 picker for any payload type.

The first three surface only when there is at least one stream payload
involved; existing client code that uses generated parameter types
with `struct`/`[]byte` payloads is unaffected by those.

| Delta | Pre-v0.30 (master) | v0.30 |
|---|---|---|
| Body payload's `ContentType()` | not consulted; picker's `mediaType` is sent | when the payload satisfies [`runtime.ContentTyper`](https://pkg.go.dev/github.com/go-openapi/runtime#ContentTyper), its non-empty return value becomes the wire `Content-Type` |
| Stage-2 octet-stream upgrade | absent; the picker's choice is the only signal | when the payload is a stream and lacks an explicit declaration, `application/octet-stream` from the operation's `consumes` list is used in preference to a structural mime like `application/json` |
| `SetHeaderParam("Content-Type", X)` | silently overwritten by `buildHTTP` | honoured at top priority; the user's explicit assertion wins |
| Stage-1 producer-capability filter | picker returns the first non-empty entry; if no producer is registered for it, the gate at the call site errors | picker skips entries with no registered producer (and no structural status) and tries the next one; only errors when nothing in `consumes` is registered |

Each delta is verified by a row in the behavioural harness at
[`client/content_negotiation_test.go`](../client/content_negotiation_test.go).
The rows that fail when the harness runs against the v0.29 baseline
are exactly the rows that exercise these three deltas — there are no
incidental behaviour changes outside this set. The structural paths
(form, multipart, file uploads) and the multipart-vs-urlencoded
preference fix from #286 are preserved verbatim.

#### Migration notes

- **No action needed** for callers using `struct`-typed parameters
  generated by go-swagger. The wire `Content-Type` is unchanged.
- **Streams that need a specific MIME type** can implement
  [`runtime.ContentTyper`](https://pkg.go.dev/github.com/go-openapi/runtime#ContentTyper)
  on the payload value, or add `application/octet-stream` to the
  operation's `consumes`, or fall back to setting the header
  explicitly via the params writer.
- **Callers that relied on `SetHeaderParam("Content-Type", …)` and
  found it didn't work** (it never did, on body requests) can now
  rely on it as a documented escape hatch for stream payloads.

## Client side — inbound responses

The client uses the operation's `Reader` plus the per-MIME `Consumers`
map. There is no `Accept` negotiation step on the client beyond the
header value the user (or codegen) sets on the request — the response
content type is taken from `Content-Type` on the response and dispatched
to the matching consumer.

## `Accept-Encoding`

[`negotiate.ContentEncoding(r, offers)`](https://pkg.go.dev/github.com/go-openapi/runtime/server-middleware/negotiate#ContentEncoding)
implements `Accept-Encoding` negotiation against a list of offered
encoding tokens (`gzip`, `deflate`, …). Encoding tokens have no
parameters, so the v0.30 parameter-honouring change does not apply.

The runtime itself does not transparently encode response bodies; this
helper is for handlers that want to make the choice explicitly.

## Common gotchas

**"My matching test broke after upgrading to v0.30."**
Likely the parameter-honouring change. If your `Accept` clients and
your `produces` use mismatched charset/version params and you treat
those as informational, opt out with `negotiate.WithIgnoreParameters(true)`
(per call) or `Context.SetIgnoreParameters(true)` (server-wide).

**"My client request returns 415 even though the API lists my type in `consumes`."**
Check the wire `Content-Type` against your server's `consumes` matching
rules. The client sends the picker's choice (with Stage-2 upgrades for
streams), so a stray space, missing charset, or trailing `;` in the
spec entry will be sent through and rejected by a strict server. If
the payload is a stream, consider implementing `ContentType() string`
on it to declare the type explicitly.

**"My stream payload's wire `Content-Type` is wrong."**
Four cases in priority order: set the header explicitly via
`SetHeaderParam("Content-Type", …)` in your params writer; implement
`runtime.ContentTyper` (`ContentType() string`) on the payload to
declare an explicit type; add `application/octet-stream` to the
operation's `consumes` list to trigger the Stage-2 upgrade; or list
the desired mime first in `consumes` so the picker chooses it.

**"My server returns 400 for a missing `Content-Type` on a body request."**
It shouldn't — missing headers fall through to `application/octet-stream`
via `runtime.DefaultMime` and that produces 415, not 400. A 400 means
the header is *present and unparseable*. Check for stray characters
(unmatched parens, wildcards in parameter names, etc.).

**"How do I get the parsed `Content-Type` value in my handler?"**
Use [`runtime.ContentType(r.Header)`](https://pkg.go.dev/github.com/go-openapi/runtime#ContentType)
or the cached value at `middleware.MatchedRouteFrom(r).Consumes`.

## Reference

- Server matching primitive: `github.com/go-openapi/runtime/server-middleware/mediatype`
- Server negotiator: `github.com/go-openapi/runtime/server-middleware/negotiate`
- Server validation: `middleware/validation.go` (`validateContentType`)
- Client Stage-1 picker: `client/runtime.go` (`pickConsumesMediaType`)
- Client Stage-2 fallback: `client/request.go` (`setStreamContentType`,
  `streamFallbackMime`, `payloadContentType`)
- Behavioural test harness: `client/content_negotiation_test.go`
- RFC 7231 §3.1.1 (media type), §5.3.1 (q-values), §5.3.2 (Accept).
