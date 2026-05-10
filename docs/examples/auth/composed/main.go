// SPDX-License-Identifier: Apache-2.0

// Command composed backs the snippets on the doc-site
// "Composed schemes (AND / OR)" recipe page
// (usage/examples/auth/composed.md).
//
// The wiring below shows how to register multiple authenticators on a
// single untyped API so that the spec can compose them with AND (inside
// one security entry) and OR (between entries). All callbacks return the
// same principal type so the operation handler need not branch on which
// scheme matched. The JWT and database lookups are stubbed; swap them
// for your own helpers in production code.
//
// `go run .` exercises the demo wiring against a no-op spec.
package main

import (
	"context"
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime/middleware/untyped"
	"github.com/go-openapi/runtime/security"
)

// --- Stubs (excluded from rendered snippets) ------------------------

// doc stands in for a real `*loads.Document` loaded via `loads.Spec`.
var doc *loads.Document

// principal is the common type returned by every authenticator. The
// runtime hands the principal of the *winning* security entry to the
// operation handler regardless of which schemes participated, so all
// callbacks must agree on this type.
type principal struct {
	Subject string
	Roles   []string
	// Source records which scheme produced this principal, useful when
	// the operation handler wants to branch on the auth flavour.
	Source string
}

// authenticateBasic stands in for a real user-store lookup behind the
// `isRegistered` Basic scheme.
func authenticateBasic(ctx context.Context, _, _ string) (context.Context, any, error) {
	return ctx, &principal{Source: "basic"}, nil
}

// verifyResellerToken stands in for the JWT parser/validator used by
// both `isReseller` (header carrier) and `isResellerQuery` (query
// carrier).
func verifyResellerToken(_ string) (any, error) {
	return &principal{Source: "reseller"}, nil
}

// verifyBearerWithScopes stands in for the OAuth2 bearer validator that
// also enforces the operation's required scopes.
func verifyBearerWithScopes(_ string, requiredScopes []string) (any, error) {
	if len(requiredScopes) == 0 {
		return nil, errors.New(http.StatusForbidden, "insufficient_scope")
	}
	return &principal{Source: "bearer", Roles: requiredScopes}, nil
}

// use silences "declared and not used" diagnostics for snippet-local
// values that exist only to make the demo compile.
func use(_ ...any) {}

func main() {
	wireComposedAuth()
}

// --- Snippets -------------------------------------------------------

func wireComposedAuth() {
	// snippet:wireComposedAuth
	api := untyped.NewAPI(doc).WithJSONDefaults()

	// One callback per scheme.
	api.RegisterAuth("isRegistered", security.BasicAuthCtx(authenticateBasic))
	api.RegisterAuth("isReseller", security.APIKeyAuth("X-Custom-Key", "header", verifyResellerToken))
	api.RegisterAuth("isResellerQuery", security.APIKeyAuth("CustomKeyAsQuery", "query", verifyResellerToken))
	api.RegisterAuth("hasRole", security.BearerAuth("hasRole", verifyBearerWithScopes))

	api.RegisterAuthorizer(security.Authorized()) // gating happens inside the authenticators
	// endsnippet:wireComposedAuth

	use(api)
}
