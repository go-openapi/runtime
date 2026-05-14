// SPDX-License-Identifier: Apache-2.0

// Command basic backs the snippets on the doc-site
// "HTTP Basic" auth recipe page. Each function below is the source of a
// `{{< code region="..." >}}` include; the package as a whole compiles
// and lints so the snippets cannot rot silently.
//
// `go run .` exercises the non-blocking demos (wiring registration).
package main

import (
	"context"
	"crypto/subtle"
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime/middleware/untyped"
	"github.com/go-openapi/runtime/security"
)

// --- Stubs (excluded from rendered snippets) ------------------------

// doc is a placeholder OpenAPI document. Snippets pretend it was loaded
// from disk; the demo wires a freshly-constructed empty spec so the
// program compiles and runs.
var doc *loads.Document

// fakePrincipal stands in for whatever the application returns from
// authentication (e.g. *models.Principal).
type fakePrincipal struct{ Name string }

// fakeStore stands in for an application-supplied user store.
type fakeStore struct{}

func (fakeStore) AuthenticateBasic(_ context.Context, user, pass string) (*fakePrincipal, error) {
	// subtle.ConstantTimeCompare avoids leaking the expected password
	// byte-by-byte via response timing. The username is non-secret and
	// compared with `==` purely to short-circuit unknown accounts.
	if user == "alice" && subtle.ConstantTimeCompare([]byte(pass), []byte("s3cret")) == 1 {
		return &fakePrincipal{Name: user}, nil
	}
	return nil, errors.Unauthenticated("basic")
}

var store = fakeStore{}

// use silences "declared and not used" diagnostics for snippet-local
// values that exist only to make the demo compile.
func use(_ ...any) {}

func main() {
	registerBasicAuth()
	failedBasicAuthChallenge()
}

// --- Snippets -------------------------------------------------------

func registerBasicAuth() {
	// snippet:registerBasicAuth
	api := untyped.NewAPI(doc).WithJSONDefaults()

	api.RegisterAuth("basicAuth", security.BasicAuthRealmCtx(
		"petstore",
		func(ctx context.Context, user, pass string) (context.Context, any, error) {
			// request-scoped lookup — honours ctx cancellation
			principal, err := store.AuthenticateBasic(ctx, user, pass)
			if err != nil {
				return ctx, nil, errors.Unauthenticated("basic")
			}
			return ctx, principal, nil
		},
	))
	// endsnippet:registerBasicAuth

	use(api)
}

func failedBasicAuthChallenge() {
	api := untyped.NewAPI(doc).WithJSONDefaults()

	// snippet:failedBasicAuthChallenge
	api.ServeError = func(w http.ResponseWriter, r *http.Request, err error) {
		if realm := security.FailedBasicAuthCtx(r.Context()); realm != "" {
			w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
		}
		errors.ServeError(w, r, err)
	}
	// endsnippet:failedBasicAuthChallenge

	use(api)
}
