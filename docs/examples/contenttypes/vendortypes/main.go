// SPDX-License-Identifier: Apache-2.0

// Command vendortypes backs the snippets on the doc-site
// "Vendor MIME types" page. Each function below is the source of a
// `{{< code region="..." >}}` include; the package as a whole compiles
// and lints so the snippets cannot rot silently.
//
// `go run .` exercises the non-blocking demos (no network is touched —
// the demos only register codecs and dispatch on a fake request).
package main

import (
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware/untyped"
)

// --- Stubs (excluded from rendered snippets) ------------------------

var doc *loads.Document

// handleV1 / handleV2 stand in for the per-version handler bodies the
// caller would supply. They return an empty payload so the snippet
// type-checks without dragging in a domain model.
func handleV1(_ any) (any, error) { return struct{}{}, nil }
func handleV2(_ any) (any, error) { return struct{}{}, nil }

// use silences "declared and not used" diagnostics for snippet-local
// values that exist only to make the demo compile.
func use(_ ...any) {}

func main() {
	registerVendorTypes()
	dispatchOnContentType()
}

// --- Snippets -------------------------------------------------------

func registerVendorTypes() {
	// snippet:registerVendorTypes
	api := untyped.NewAPI(doc).WithJSONDefaults()

	api.RegisterConsumer("application/vnd.acme.v1+json", runtime.JSONConsumer())
	api.RegisterProducer("application/vnd.acme.v1+json", runtime.JSONProducer())

	api.RegisterConsumer("application/vnd.acme.v2+json", runtime.JSONConsumer())
	api.RegisterProducer("application/vnd.acme.v2+json", runtime.JSONProducer())
	// endsnippet:registerVendorTypes

	use(api)
}

func dispatchOnContentType() {
	// snippet:dispatchOnContentType
	handlePost := func(r *http.Request, body any) (any, error) {
		ct, _, _ := runtime.ContentType(r.Header)
		switch ct {
		case "application/vnd.acme.v1+json":
			return handleV1(body)
		case "application/vnd.acme.v2+json":
			return handleV2(body)
		}
		return nil, errors.New(http.StatusUnsupportedMediaType, "unsupported version")
	}
	// endsnippet:dispatchOnContentType

	use(handlePost)
}
