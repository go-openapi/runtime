// SPDX-License-Identifier: Apache-2.0

// Command contenttypes backs the snippets on the doc-site
// "Content types" page. Each function below is the source of a
// `{{< code region="..." >}}` include; the package as a whole compiles
// and lints so the snippets cannot rot silently.
//
// `go run .` exercises the non-blocking demos (codec registration).
package main

import (
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/client"
	"github.com/go-openapi/runtime/middleware/untyped"
)

// --- Stubs (excluded from rendered snippets) ------------------------

// spec is a placeholder OpenAPI document. Snippets pretend it was
// loaded from disk; the demo wires a nil spec so the program compiles
// and runs.
var spec *loads.Document

// use silences "declared and not used" diagnostics for snippet-local
// values that exist only to make the demo compile.
func use(_ ...any) {}

func main() {
	registerCodecsServer()
	registerCodecsClient()
}

// --- Snippets -------------------------------------------------------

func registerCodecsServer() {
	// snippet:registerCodecsServer
	api := untyped.NewAPI(spec)
	api.RegisterConsumer(runtime.JSONMime, runtime.JSONConsumer())
	api.RegisterProducer(runtime.JSONMime, runtime.JSONProducer())
	api.RegisterConsumer("application/vnd.acme.v1+json", runtime.JSONConsumer())
	api.RegisterProducer("application/vnd.acme.v1+json", runtime.JSONProducer())
	// endsnippet:registerCodecsServer

	use(api)
}

func registerCodecsClient() {
	// snippet:registerCodecsClient
	rt := client.New("api.example.com", "/v1", []string{"https"})
	rt.Consumers[runtime.JSONMime] = runtime.JSONConsumer()
	rt.Producers[runtime.JSONMime] = runtime.JSONProducer()
	// endsnippet:registerCodecsClient

	use(rt)
}
