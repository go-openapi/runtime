// SPDX-License-Identifier: Apache-2.0

// Command apikey backs the snippets on the doc-site
// "API key (single scheme)" example page. The `wireAPIKeyAuth`
// region below is the source for the `{{< code region="..." >}}`
// include; the package as a whole compiles and lints so the
// snippet cannot rot silently.
//
// `go run .` is a no-op (the demo is built around a blocking HTTP
// listener). Pass `serve` to actually start the server on :35307.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/runtime/middleware/untyped"
	"github.com/go-openapi/runtime/security"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		wireAPIKeyAuth()
	}
}

// --- Snippets -------------------------------------------------------

func wireAPIKeyAuth() {
	// snippet:wireAPIKeyAuth
	doc, _ := loads.Spec("swagger.yml")
	api := untyped.NewAPI(doc).WithJSONDefaults()

	// 1. Authenticator: token → principal
	api.RegisterAuth("key", security.APIKeyAuth(
		"X-Token", "header",
		func(token string) (any, error) {
			if token == "abcdefuvwxyz" {
				return "alice", nil
			}
			return nil, errors.New(http.StatusUnauthorized, "invalid api key")
		},
	))

	// 2. Authorizer: every authenticated principal allowed.
	//    (Skip this line if you have no business-rule gating.)
	api.RegisterAuthorizer(security.Authorized())

	// 3. Operation handlers (one per spec operation)
	api.RegisterOperation("get", "/customers/{id}", runtime.OperationHandlerFunc(
		func(_ any) (any, error) {
			// params is the bound parameter struct;
			// principal is on r.Context() via middleware.SecurityPrincipalFrom
			return map[string]string{"id": "42"}, nil
		},
	))

	handler := middleware.Serve(doc, api)
	log.Fatal(http.ListenAndServe(":35307", handler)) //nolint:gosec // demo handler, no timeouts needed
	// endsnippet:wireAPIKeyAuth
}
