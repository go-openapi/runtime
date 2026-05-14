// SPDX-License-Identifier: Apache-2.0

// Command docui backs the snippets on the doc-site
// "Doc UIs & spec serving" page. Each function below is the source of a
// `{{< code region="..." >}}` include; the package as a whole compiles
// and lints so the snippets cannot rot silently.
//
// `go run .` exits immediately (no demos run by default). Pass `serve`
// to run the full Swagger UI + spec demo (which blocks on
// ListenAndServe).
package main

import (
	_ "embed"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-openapi/runtime/server-middleware/docui"
)

// --- Stubs (excluded from rendered snippets) ------------------------

// myAPIHandler returns the demo application handler used by the
// snippets. It is intentionally minimal — the snippets only care that
// something implementing http.Handler is available.
func myAPIHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/ping", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("pong"))
	})
	return mux
}

// use silences "declared and not used" diagnostics for snippet-local
// values that exist only to make the demo compile.
func use(_ ...any) {}

const readHeaderTimeout = 5 * time.Second

//go:embed swagger.json
var specBytes []byte

func main() {
	directWrap(false)
	middlewareFactory()
	serveSpec()
	useSpec()
	pathFromOptions()
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		puttingItTogether()
	}
}

// --- Snippets -------------------------------------------------------

func directWrap(listen bool) {
	// snippet:directWrap
	api := myAPIHandler() // your application

	handler := docui.SwaggerUI(api,
		docui.WithSpecURL("/swagger.json"),
		docui.WithUIBasePath("/"),
		docui.WithUIPath("docs"),
	)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
	}
	// endsnippet:directWrap

	if listen {
		log.Fatal(srv.ListenAndServe())
	}
	use(srv)
}

func middlewareFactory() {
	api := myAPIHandler()
	mux := http.NewServeMux()

	// snippet:middlewareFactory
	mw := docui.UseSwaggerUI(
		docui.WithSpecURL("/swagger.json"),
		docui.WithUIPath("docs"),
	)
	mux.Handle("/", mw(api))
	// endsnippet:middlewareFactory

	use(mux)
}

func serveSpec() {
	api := myAPIHandler()

	// snippet:serveSpec
	handler := docui.ServeSpec(specBytes, api,
		docui.WithSpecPath("/swagger.json"),
	)
	// endsnippet:serveSpec

	use(handler)
}

func useSpec() {
	api := myAPIHandler()
	mux := http.NewServeMux()

	// snippet:useSpec
	mw := docui.UseSpec(specBytes,
		docui.WithSpecPath("/swagger.json"),
	)
	mux.Handle("/", mw(api))
	// endsnippet:useSpec

	use(mux)
}

func pathFromOptions() {
	api := myAPIHandler()

	// snippet:pathFromOptions
	uiOpts := []docui.Option{docui.WithSpecURL("/swagger.json")}
	specOpt := docui.WithSpecPathFromOptions(uiOpts...)

	handler := docui.SwaggerUI(
		docui.ServeSpec(specBytes, api, specOpt),
		uiOpts...,
	)
	// endsnippet:pathFromOptions

	use(handler)
}

// snippet:puttingItTogether

//go:embed openapi.yaml
var spec []byte

func puttingItTogether() {
	api := http.NewServeMux()
	api.HandleFunc("/v1/ping", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("pong"))
	})

	handler := docui.SwaggerUI(
		docui.ServeSpec(spec, api,
			docui.WithSpecPath("/openapi.yaml"),
		),
		docui.WithSpecURL("/openapi.yaml"),
		docui.WithUIPath("docs"),
		docui.WithUITitle("Demo API"),
	)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
	}
	log.Fatal(srv.ListenAndServe())
}

// endsnippet:puttingItTogether
