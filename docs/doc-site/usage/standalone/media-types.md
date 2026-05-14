---
title: Media types
weight: 10
description: |
  Typed RFC 7231 media-type values, sets and asymmetric matching via
  the mediatype package.
---

[`server-middleware/mediatype`](https://pkg.go.dev/github.com/go-openapi/runtime/server-middleware/mediatype)
provides the parsed value type, the matching rule and the helper used by
both server-side `Content-Type` validation and `Accept`-header
negotiation.

## The `MediaType` value

```go
type MediaType struct {
    Type    string             // lowercased on parse
    Subtype string             // lowercased on parse
    Params  map[string]string  // keys lowercased; values verbatim
    Q       float64            // extracted from "q="; not stored in Params
}
```

See
[`MediaType`](https://pkg.go.dev/github.com/go-openapi/runtime/server-middleware/mediatype#MediaType)
on pkg.go.dev for the authoritative definition.

[`Parse`](https://pkg.go.dev/github.com/go-openapi/runtime/server-middleware/mediatype#Parse)
accepts a single media type:

{{< code file="standalone/mediatypes/main.go" lang="go" region="parseMediaType" >}}

Parameter **values** are preserved verbatim, but comparisons are
case-insensitive (`charset=UTF-8` matches `charset=utf-8`). Wildcards
`*/*` and `type/*` are accepted on either side; `*/subtype` is invalid
and `Parse` rejects it.

### Specificity

`MediaType.Specificity()` returns one of the constants below — useful
when writing custom selection logic:

| Constant                     | Example                       |
|------------------------------|-------------------------------|
| `SpecificityAny`             | `*/*`                         |
| `SpecificityType`            | `text/*`                      |
| `SpecificityExact`           | `text/plain`                  |
| `SpecificityExactWithParams` | `text/plain;charset=utf-8`    |

## The asymmetric matching rule

`MediaType.Matches(other)` is **asymmetric**. The receiver is the *bound*
(an allowed entry on the server side, or a candidate offer when matching
against an `Accept` entry); the argument is the *constraint* (the actual
request value, or the `Accept` entry being satisfied).

The rule:

1. Bare `type/subtype` must agree (with wildcards on either side).
2. If the receiver carries **no parameters**, any constraint is accepted
   regardless of its parameters.
3. Otherwise every `(key, value)` pair on the constraint must be present
   on the receiver, with case-insensitive value comparison. The receiver
   may carry additional parameters that the constraint does not list.

q-values are not considered by `Matches` — they belong to the negotiator
(see [`Set.BestMatch`](#sets-and-bestmatch)).

The same direction is used in both call sites in the runtime:

| Call                  | Bound (receiver)         | Constraint (argument)    |
|-----------------------|--------------------------|--------------------------|
| Inbound validation    | each entry in `consumes` | request's `Content-Type` |
| `Accept` negotiation  | each candidate offer     | each `Accept` entry      |

## `MatchFirst` — the validation primitive

```go
func MatchFirst(allowed []string, actual string) (MediaType, bool, error)
```

See
[`MatchFirst`](https://pkg.go.dev/github.com/go-openapi/runtime/server-middleware/mediatype#MatchFirst)
on pkg.go.dev for the authoritative signature.

Used when you need a yes/no answer plus the matched bound. Short-circuits
on the first allowed entry that accepts `actual` (so the returned
`MediaType` is **not** necessarily the most specific match — use
`Set.BestMatch` if you need ranked selection).

| Return                       | Meaning                                                                                       |
|------------------------------|-----------------------------------------------------------------------------------------------|
| `(matched, true,  nil)`      | first allowed entry that accepts `actual`                                                     |
| `(zero,    false, nil)`      | `actual` is well-formed but no allowed entry accepts it (HTTP 415 territory)                  |
| `(zero,    false, err)`      | `actual` failed to parse; `err` wraps `ErrMalformed` (HTTP 400 territory — `errors.Is` it)    |

Allowed entries that themselves fail to parse are skipped silently
(they cannot match a well-formed actual).

## Sets and `BestMatch`

```go
type Set []MediaType

func ParseAccept(s string) Set
func (s Set) BestMatch(offered Set) (best MediaType, ok bool)
```

See
[`Set`](https://pkg.go.dev/github.com/go-openapi/runtime/server-middleware/mediatype#Set),
[`ParseAccept`](https://pkg.go.dev/github.com/go-openapi/runtime/server-middleware/mediatype#ParseAccept)
and
[`Set.BestMatch`](https://pkg.go.dev/github.com/go-openapi/runtime/server-middleware/mediatype#Set.BestMatch)
on pkg.go.dev for the authoritative signatures.

`ParseAccept` parses a comma-separated list (e.g. an `Accept` header
value), skipping malformed entries silently — be liberal in what you
accept.

`BestMatch` ranks the *offered* set against the receiver `Accept` set:

1. Highest q-value wins.
2. Ties on q broken by the highest `Specificity` of the matching `Accept` entry.
3. Ties on specificity broken by earliest position in `offered`.

Accept entries with `q=0` are treated as **exclusions** and never match.
Returns `ok=false` if no offer matched any non-zero-q entry.

For the full algorithm — including how `negotiate` wires this up and the
v0.30 parameter-honouring change — see
[tutorials / media-type selection](../../tutorials/media-types/)
and the next page,
[content negotiation](../content-negotiation/).
