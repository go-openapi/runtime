// SPDX-License-Identifier: Apache-2.0

// Command requests backs the snippets on the doc-site
// "Building & submitting requests" page. Each function below is the source of a
// `{{< code region="..." >}}` include; the package as a whole compiles
// and lints so the snippets cannot rot silently.
//
// `go run .` exercises the non-blocking demos. The snippets here construct
// requests against a fake host — nothing is actually sent over the wire.
package main

import (
	"context"
	"net/http"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/client"
)

// --- Stubs (excluded from rendered snippets) ------------------------

// rt is a placeholder client transport. Snippets pretend it was configured
// against a real host; the demo wires a freshly-constructed Runtime so the
// program compiles and runs.
var rt = client.New("example.invalid", "/", []string{"https"})

// op is a placeholder operation descriptor. Snippets pretend it was built
// by a generated client; the demo wires a zero-valued descriptor with the
// minimum required fields so the program compiles and runs.
var op = &runtime.ClientOperation{
	ID:          "demo",
	Method:      http.MethodGet,
	PathPattern: "/",
	Reader: runtime.ClientResponseReaderFunc(func(_ runtime.ClientResponse, _ runtime.Consumer) (any, error) {
		return struct{}{}, nil
	}),
}

// parent is a placeholder parent context.
var parent = context.Background()

// myClient is a placeholder http client used by the build-only snippet.
var myClient = http.DefaultClient

// use silences "declared and not used" diagnostics for snippet-local
// values that exist only to make the demo compile.
func use(_ ...any) {}

const exampleTimeout = 5 * time.Second

func main() {
	submitVariants()
	resp, err := createHTTPRequest()
	if resp != nil {
		defer resp.Body.Close()
	}
	use(resp, err)
	migrationForm()
}

// --- Snippets -------------------------------------------------------

func submitVariants() {
	// snippet:submitVariants
	// legacy — cached context, hard to cancel from the call site
	result, err := rt.Submit(op)
	use(result, err)

	// preferred — explicit context
	ctx, cancel := context.WithTimeout(parent, exampleTimeout)
	defer cancel()
	result, err = rt.SubmitContext(ctx, op)
	// endsnippet:submitVariants

	use(result, err)
}

func createHTTPRequest() (*http.Response, error) {
	ctx := context.Background()

	// snippet:createHTTPRequestContext
	req, cancel, err := rt.CreateHTTPRequestContext(ctx, op)
	if err != nil {
		return nil, err
	}
	defer cancel() // MUST run after the response is fully read

	resp, err := myClient.Do(req)
	// ...
	// endsnippet:createHTTPRequestContext
	return resp, err
}

func migrationForm() {
	ctx := context.Background()

	// snippet:migrationForm
	// before
	op.Context = ctx
	result, err := rt.Submit(op)
	use(result, err)

	// after
	result, err = rt.SubmitContext(ctx, op)
	// endsnippet:migrationForm

	use(result, err)
}
