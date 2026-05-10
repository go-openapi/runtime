// SPDX-License-Identifier: Apache-2.0

// Command bearerjwt backs the snippets on the doc-site
// "Bearer + JWT" recipe page (usage/examples/auth/bearer-jwt.md).
//
// The wiring below shows how to register a Bearer authenticator on an
// untyped API and intersect the JWT's claimed roles with the operation's
// required scopes. JWT parsing is stubbed (`parseJWT`) so the example
// does not pull a specific JWT library into the doc-examples module;
// swap it for `jwt.ParseWithClaims` from `github.com/golang-jwt/jwt/v5`
// (or any other parser) in your own code.
//
// `go run .` exercises the demo wiring against a no-op spec.
package main

import (
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime/middleware/untyped"
	"github.com/go-openapi/runtime/security"
)

// --- Stubs (excluded from rendered snippets) ------------------------

// doc stands in for a real `*loads.Document` loaded via `loads.Spec`.
var doc *loads.Document

// principal is what the authenticator returns on success. The runtime
// stores it in the request context for the operation handler to use.
type principal struct {
	Subject string
	Roles   []string
}

// roleClaims is the subset of JWT claims the example cares about.
// In real code this would embed `jwt.RegisteredClaims`.
type roleClaims struct {
	Subject string
	Roles   []string
}

// parseJWT is a stand-in for a real JWT parser. Replace with
// `jwt.ParseWithClaims(token, &roleClaims{}, keyFn)` (or your introspection
// call) in production code.
func parseJWT(_ string) (*roleClaims, error) {
	return &roleClaims{}, nil
}

// intersect returns the elements present in both slices. A real
// implementation would normalise case and dedupe.
func intersect(a, b []string) []string {
	set := make(map[string]struct{}, len(b))
	for _, s := range b {
		set[s] = struct{}{}
	}
	out := make([]string, 0, len(a))
	for _, s := range a {
		if _, ok := set[s]; ok {
			out = append(out, s)
		}
	}
	return out
}

// use silences "declared and not used" diagnostics for snippet-local
// values that exist only to make the demo compile.
func use(_ ...any) {}

func main() {
	wireBearerAuth()
}

// --- Snippets -------------------------------------------------------

func wireBearerAuth() {
	// snippet:wireBearerAuth
	api := untyped.NewAPI(doc).WithJSONDefaults()

	api.RegisterAuth("hasRole", security.BearerAuth("hasRole",
		func(token string, requiredScopes []string) (any, error) {
			claims, err := parseJWT(token)
			if err != nil {
				return nil, errors.Unauthenticated("bearer")
			}

			// intersect claimed roles with required scopes
			granted := intersect(claims.Roles, requiredScopes)
			if len(granted) == 0 {
				return nil, errors.New(http.StatusForbidden, "insufficient_scope")
			}

			return &principal{
				Subject: claims.Subject,
				Roles:   granted,
			}, nil
		},
	))
	// endsnippet:wireBearerAuth

	use(api)
}
