// SPDX-License-Identifier: Apache-2.0

// Command deprecatedshims backs the snippets on the doc-site
// "Deprecated shims" page. Each function below is the source of a
// `{{< code region="..." >}}` include; the package as a whole compiles
// and lints so the snippets cannot rot silently.
//
// The page shows the old (deprecated) middleware entry points side-by-
// side with their new server-middleware equivalents. Both halves are
// exercised here so neither path can rot.
//
// `go run .` exercises the non-blocking demos.
package main

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/runtime/server-middleware/docui"
	"github.com/go-openapi/runtime/server-middleware/negotiate"
)

// --- Stubs (excluded from rendered snippets) ------------------------

// api is a placeholder downstream handler that the doc-UI middleware
// wraps. Snippets pretend it is the application's API mux.
var api = http.NotFoundHandler()

// newRequest builds a throwaway *http.Request with an Accept header so
// the negotiation snippets have something to chew on.
func newRequest() *http.Request {
	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	r.Header.Set("Accept", "application/json")
	return r
}

// use silences "declared and not used" diagnostics for snippet-local
// values that exist only to make the demo compile.
func use(_ ...any) {}

func main() {
	negotiateBefore()
	negotiateAfter()
	swaggerUIBefore()
	swaggerUIAfter()
}

// --- Snippets -------------------------------------------------------

func negotiateBefore() {
	r := newRequest()
	offers := []string{"application/json", "application/xml"}

	// snippet:negotiateBefore
	// before
	// import "github.com/go-openapi/runtime/middleware"
	chosen := middleware.NegotiateContentType(r, offers, "application/json") //nolint:staticcheck // intentionally demonstrating deprecated API
	// endsnippet:negotiateBefore

	use(chosen)
}

func negotiateAfter() {
	r := newRequest()
	offers := []string{"application/json", "application/xml"}

	// snippet:negotiateAfter
	// after
	// import "github.com/go-openapi/runtime/server-middleware/negotiate"
	chosen := negotiate.ContentType(r, offers, "application/json")
	// endsnippet:negotiateAfter

	use(chosen)
}

func swaggerUIBefore() {
	// snippet:swaggerUIBefore
	// before
	// import "github.com/go-openapi/runtime/middleware"

	handler := middleware.SwaggerUI(middleware.SwaggerUIOpts{ //nolint:staticcheck // intentionally demonstrating deprecated API
		BasePath: "/",
		Path:     "docs",
		SpecURL:  "/swagger.json",
		Title:    "Pet store",
	}, api)
	// endsnippet:swaggerUIBefore

	use(handler)
}

func swaggerUIAfter() {
	// snippet:swaggerUIAfter
	// after
	// import "github.com/go-openapi/runtime/server-middleware/docui"

	handler := docui.SwaggerUI(api,
		docui.WithUIBasePath("/"),
		docui.WithUIPath("docs"),
		docui.WithSpecURL("/swagger.json"),
		docui.WithUITitle("Pet store"),
	)
	// endsnippet:swaggerUIAfter

	use(handler)
}
