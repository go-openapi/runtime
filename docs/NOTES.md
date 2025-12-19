### v0.29.0

**New with this release**:

* upgraded to `go1.24` and modernized the code base accordingly
* updated all dependencies, and removed an noticable indirect dependency (e.g. `mailru/easyjson`)
* **breaking change** no longer imports `opentracing-go` (#365).
    * the `WithOpentracing()` method now returns an opentelemetry transport
    * for users who can't transition to opentelemetry, the previous behavior
      of `WithOpentracing` delivering an opentracing transport is provided by a separate
      module `github.com/go-openapi/runtime/client-middleware/opentracing`.
* removed direct dependency to `gopkg.in/yaml.v3`, in favor of `go.yaml.in/yaml/v3` (an indirect
  test dependency to the older package is still around)
* technically, the repo has evolved to a mono-repo, multiple modules structures (2 go modules
  published), with CI adapted accordingly
