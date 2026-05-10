// SPDX-License-Identifier: Apache-2.0

// Command intro backs the snippet on the doc-site "Client" landing page.
// The function below is the source of a `{{< code region="..." >}}`
// include; the package as a whole compiles and lints so the snippet
// cannot rot silently.
//
// `go run .` exercises the demo (the SubmitContext call is expected to
// fail against the placeholder host — wiring is what we demonstrate).
package main

import (
	"context"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/client"
)

// --- Stubs (excluded from rendered snippets) ------------------------

// token stands in for an OAuth2 / bearer token sourced from the caller's
// secret store.
var token = "demo-token"

// op stands in for a *runtime.ClientOperation that go-swagger-generated
// clients build for each operation. The zero value is enough to make
// the snippet compile; it will not produce a working HTTP request.
var op = &runtime.ClientOperation{}

// use silences "declared and not used" diagnostics for snippet-local
// values that exist only to make the demo compile.
func use(_ ...any) {}

func main() {
	minimalClient(context.Background())
}

// --- Snippets -------------------------------------------------------

func minimalClient(ctx context.Context) {
	// snippet:minimalClient
	rt := client.New("api.example.com", "/v1", []string{"https"})
	rt.DefaultAuthentication = client.BearerToken(token)

	result, err := rt.SubmitContext(ctx, op)
	// endsnippet:minimalClient

	use(rt, result, err)
}
