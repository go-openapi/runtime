# Track B.2 — Extract `docui` (execution plan)

Status: in progress.
Scope: move the doc-UI middlewares (`SwaggerUI`, `Redoc`, `RapiDoc`,
`SwaggerUIOAuth2Callback`, `Spec`) and their option helpers into a new
`server-middleware/docui` package, leaving deprecated forwarders behind.

Parent: `roadmap-media-and-modularization.md` Track B.2.

---

## Module layout

Per fred's note in roadmap §B.4 ("most middlewares only depend on stdlib,
perhaps just 1 module is enough to hold them"), the umbrella is **one** Go
module:

```
server-middleware/
├── go.mod                                    // module: github.com/go-openapi/runtime/server-middleware
├── go.sum
└── docui/                                    // package: docui
    ├── doc.go
    ├── options.go        // UIOptions, UIOption, WithXxx, EnsureDefaults helpers
    ├── render.go         // serveUI helper (private)
    ├── swaggerui.go
    ├── swaggerui_oauth2.go
    ├── redoc.go
    ├── rapidoc.go
    ├── spec.go           // ServeSpec, SpecOption, WithSpecPath, WithSpecDocument
    └── *_test.go
```

`go.work` adds `./server-middleware` to its `use` clause.

## Cross-module dependency

`docui` is stdlib-only (production code). `runtime` will gain a runtime
dependency on `server-middleware/` for `middleware/context.go`. We follow
the established pattern from `client-middleware/opentracing`:

```go
// runtime/go.mod
require github.com/go-openapi/runtime/server-middleware v0.30.0
replace github.com/go-openapi/runtime/server-middleware => ./server-middleware
```

Both modules will release together at v0.30.0 (per fred's roadmap note on
versioning).

## Exported surface of `docui`

| Name (in docui) | Origin | Notes |
|-----------------|--------|-------|
| `UIOptions` | was unexported `uiOptions` | exported because `Context` needs to hold and convert it |
| `UIOption` | already exported | unchanged |
| `WithUIBasePath`, `WithUIPath`, `WithUISpecURL`, `WithUITitle`, `WithTemplate` | already exported | unchanged |
| `UIOptionsWithDefaults` | was unexported `uiOptionsWithDefaults` | exported |
| `FromCommonToAnyOptions` | was unexported `fromCommonToAnyOptions` | exported, same generic shape |
| `ToCommonUIOptions` | was unexported `toCommonUIOptions` | exported |
| `SwaggerUIOpts`, `SwaggerUI`, `SwaggerUIOAuth2Callback` | already exported | unchanged |
| `RedocOpts`, `Redoc` | already exported | unchanged |
| `RapiDocOpts`, `RapiDoc` | already exported | unchanged |
| `SpecOption`, `WithSpecPath`, `WithSpecDocument` | already exported | unchanged |
| `ServeSpec` | renamed from `Spec` | the package-name + `Spec` reads as `docui.Spec`, which is ambiguous in user code |

Asset URL constants (`swaggerLatest`, `redocLatest`, ...), HTML templates,
and helper funcs (`serveUI`, `EnsureDefaults` flavor methods,
`defaultDocsPath/URL/Title`, `contentTypeHeader`, `applicationJSON`) stay
private to `docui`.

## Backward-compat forwarders in `middleware/`

For each moved file, we leave a thin shim that aliases the type and
forwards the func:

```go
// Deprecated: moved to server-middleware/docui. Use docui.SwaggerUIOpts.
type SwaggerUIOpts = docui.SwaggerUIOpts

// Deprecated: moved to server-middleware/docui. Use docui.SwaggerUI.
var SwaggerUI = docui.SwaggerUI
```

`Spec` keeps its name in `middleware/` but forwards to `docui.ServeSpec`:

```go
// Deprecated: moved to server-middleware/docui. Use docui.ServeSpec.
var Spec = docui.ServeSpec
```

Because each old struct (`SwaggerUIOpts`, `RedocOpts`, `RapiDocOpts`) is a
plain Go type alias to its `docui` counterpart, fields and methods
(including `EnsureDefaults`) carry over identically — fully transparent
backward compat.

`middleware/context.go` updates to call `docui.SwaggerUI`, `docui.Redoc`,
`docui.RapiDoc`, `docui.ServeSpec` directly. Its private
`uiOptionsForHandler` returns `docui.UIOptions` instead of the old
unexported `uiOptions`.

## Tests

- `swaggerui_test.go`, `swaggerui_oauth2_test.go`, `redoc_test.go`,
  `rapidoc_test.go`, `ui_options_test.go` move verbatim to `docui/` (same
  package, only stdlib + testify imports).
- `spec_test.go` is rewritten for `docui` to drop the petstore fixture and
  the `runtime.HeaderContentType` literal — petstore lives in
  `runtime/internal/testing/`, which is unreachable from a sibling module.
  The new test uses raw spec bytes and the local `contentTypeHeader`
  constant.
- A small smoke test stays in `middleware/` per file, asserting that the
  deprecated alias still resolves and the forwarded handler returns 200
  on the documented path. This guards against accidental drift.

## Out of scope (deliberately)

- **Unification of `SwaggerUIOpts / RedocOpts / RapiDocOpts` into a single
  `docui.Options`** — the roadmap proposed this. Doing it now would block
  clean `type X = docui.X` aliases (the existing types differ in field
  shape) and would force every external caller through a migration.
  Defer to a follow-up issue once the move stabilises.
- **`upload`, `negotiate`, etc.** — separate Track B steps.

## Step-by-step

1. Scaffold `server-middleware/go.mod` (stdlib + testify deps for tests),
   add to `go.work`.
2. Create `server-middleware/docui/` files by moving content out of
   `middleware/{swaggerui,swaggerui_oauth2,redoc,rapidoc,spec,ui_options}.go`.
3. Export the names that need to cross the module boundary
   (`UIOptions`, `UIOptionsWithDefaults`, `FromCommonToAnyOptions`,
   `ToCommonUIOptions`) and rename `Spec` → `ServeSpec`.
4. Move tests; rewrite `spec_test.go` to drop the petstore fixture.
5. Replace each moved file in `middleware/` with a deprecation shim
   (type alias + var alias for the function).
6. Update `middleware/context.go` to call `docui.*` directly and to use
   `docui.UIOptions` in its private helper signature.
7. Update `runtime/go.mod` (require + replace) and run
   `go test ./...` in both modules.
8. Add a minimal smoke test per shim in `middleware/`.
9. `golangci-lint run --new-from-rev master` clean.

## Track B.5 - Refactor midleware options for UI

Objective: in the new package, replace the xxxOption struct argument by the more
modern function options pattern.

The xxxOption structs remain part of the deprecated "seam.go" and are not reconducted in the new module.
