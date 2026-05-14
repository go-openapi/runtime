// SPDX-License-Identifier: Apache-2.0

// Command pipeline backs the snippets on the doc-site
// "Request pipeline" page. Each function below is the source of a
// `{{< code region="..." >}}` include; the package as a whole
// compiles and lints so the snippets cannot rot silently.
//
// `go run .` exercises the non-blocking demos. Pass `serve` to run the
// untyped HTTP server demo (which blocks on ListenAndServe).
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-openapi/analysis"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/runtime/middleware/untyped"
	"github.com/justinas/alice"
)

// --- Stubs (excluded from rendered snippets) ------------------------

var (
	spec     *loads.Document
	api      *untyped.API
	myAPI    middleware.RoutableAPI
	analyzed *analysis.Spec
	doc      *loads.Document
)

var myGetPetHandler = runtime.OperationHandlerFunc(func(_ any) (any, error) {
	return struct{}{}, nil
})

func loggingMW(next http.Handler) http.Handler   { return next }
func rateLimitMW(next http.Handler) http.Handler { return next }

// use silences "declared and not used" diagnostics for snippet-local
// values that exist only to make the demo compile.
func use(_ ...any) {}

func main() {
	contextConstructors()
	aliceComposition()
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		untypedServer()
	}
}

// --- Snippets -------------------------------------------------------

func contextConstructors() {
	// snippet:contextConstructors
	// Default — untyped.API wrapped in a routableUntypedAPI.
	ctxDefault := middleware.NewContext(spec, api, nil)

	// Custom — anything that implements RoutableAPI.
	ctxCustom := middleware.NewRoutableContext(spec, myAPI, nil)

	// Same, with a pre-analyzed spec to skip re-analysis.
	ctxAnalyzed := middleware.NewRoutableContextWithAnalyzedSpec(spec, analyzed, myAPI, nil)
	// endsnippet:contextConstructors

	use(ctxDefault, ctxCustom, ctxAnalyzed)
}

const readHeaderTimeout = 5 * time.Second

func untypedServer() {
	// snippet:untypedServer
	doc, _ := loads.Spec("api.yaml")
	api := untyped.NewAPI(doc)
	api.RegisterConsumer(runtime.JSONMime, runtime.JSONConsumer())
	api.RegisterProducer(runtime.JSONMime, runtime.JSONProducer())
	api.RegisterOperation("get", "/pets/{id}", myGetPetHandler)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           middleware.Serve(doc, api),
		ReadHeaderTimeout: readHeaderTimeout,
	}
	log.Fatal(srv.ListenAndServe())
	// endsnippet:untypedServer
}

func aliceComposition() {
	// snippet:aliceComposition
	decorate := func(next http.Handler) http.Handler {
		return alice.New(loggingMW, rateLimitMW).Then(next)
	}

	handler := middleware.ServeWithBuilder(doc, api, decorate)
	// endsnippet:aliceComposition

	use(handler)
}
