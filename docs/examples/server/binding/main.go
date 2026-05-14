// SPDX-License-Identifier: Apache-2.0

// Command binding backs the snippets on the doc-site
// "Parameter binding & validation" page. Each function below is the
// source of a `{{< code region="..." >}}` include; the package as a
// whole compiles and lints so the snippets cannot rot silently.
//
// `go run .` exercises the non-blocking demos.
package main

import (
	"net/http"

	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/runtime/middleware/untyped"
)

// --- Stubs (excluded from rendered snippets) ------------------------

// spec is a placeholder OpenAPI document. Snippets pretend it was
// loaded from disk; the demo leaves it nil so the program compiles.
var (
	spec *loads.Document
	api  *untyped.API
)

// use silences "declared and not used" diagnostics for snippet-local
// values that exist only to make the demo compile.
func use(_ ...any) {}

// extraMiddleware is a placeholder Builder demonstrating how an extra
// middleware layer can read the matched route from the request
// context without rebinding.
func extraMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		readMatchedRoute(r)
		next.ServeHTTP(w, r)
	})
}

func main() {
	ignoreParameters()
	use(extraMiddleware)
}

// --- Snippets -------------------------------------------------------

func ignoreParameters() {
	// snippet:ignoreParameters
	ctx := middleware.NewContext(spec, api, nil).SetIgnoreParameters(true)
	handler := ctx.APIHandler(middleware.PassthroughBuilder)
	// endsnippet:ignoreParameters

	use(handler)
}

func readMatchedRoute(r *http.Request) {
	// snippet:readMatchedRoute
	// inside middleware.Builder
	match := middleware.MatchedRouteFrom(r)
	// (no public accessor for the bound struct itself today —
	// re-call BindValidRequest if you need it; the result is cached
	// so a second call is cheap)
	// endsnippet:readMatchedRoute

	use(match)
}
