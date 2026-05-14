// SPDX-License-Identifier: Apache-2.0

// Command customauthorizer backs the snippets on the doc-site
// "Custom Authorizer (RBAC)" recipe page. Each function below is the
// source of a `{{< code region="..." >}}` include; the package as a
// whole compiles and lints so the snippets cannot rot silently.
//
// `go run .` exercises the non-blocking demos (wiring registration).
package main

import (
	"fmt"
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/runtime/middleware/untyped"
	"github.com/go-openapi/runtime/security"
)

// --- Stubs (excluded from rendered snippets) ------------------------

// doc is a placeholder OpenAPI document. Snippets pretend it was loaded
// from disk; the demo wires a freshly-constructed empty spec so the
// program compiles.
var doc *loads.Document

// verifyBearer is a placeholder ScopedTokenAuthentication callback. A
// real implementation would validate the bearer token and return a
// principal (here a *principal carrying roles).
var verifyBearer security.ScopedTokenAuthentication = func(_ string, _ []string) (any, error) {
	return &principal{Subject: "alice", Roles: []string{"reader"}}, nil
}

// use silences "declared and not used" diagnostics for snippet-local
// values that exist only to make the demo compile.
func use(_ ...any) {}

func main() {
	_ = RBACAuthorizer()
	wireAuthorizer()
	readPrincipal(&http.Request{})
}

// --- Snippets -------------------------------------------------------

// snippet:rbacAuthorizer
type principal struct {
	Subject string
	Roles   []string
}

// roleACL: which roles may call which "method path".
var roleACL = map[string]map[string]bool{
	"GET /pets":         {"reader": true, "admin": true}, //nolint:goconst // doc example
	"POST /pets":        {"writer": true, "admin": true},
	"DELETE /pets/{id}": {"admin": true},
}

func RBACAuthorizer() runtime.Authorizer {
	return runtime.AuthorizerFunc(func(r *http.Request, p any) error {
		route := middleware.MatchedRouteFrom(r)
		key := fmt.Sprintf("%s %s", r.Method, route.PathPattern)

		allowed, ok := roleACL[key]
		if !ok {
			return errors.New(http.StatusForbidden, "no ACL entry for %s", key)
		}

		prin, ok := p.(*principal)
		if !ok {
			return errors.New(http.StatusForbidden, "principal type mismatch")
		}

		for _, role := range prin.Roles {
			if allowed[role] {
				return nil // 👍
			}
		}

		return errors.New(http.StatusForbidden, "role %v cannot %s", prin.Roles, key)
	})
}

// endsnippet:rbacAuthorizer

func wireAuthorizer() {
	// snippet:wireAuthorizer
	api := untyped.NewAPI(doc).WithJSONDefaults()

	api.RegisterAuth("bearer", security.BearerAuth("bearer", verifyBearer))
	api.RegisterAuthorizer(RBACAuthorizer())
	// endsnippet:wireAuthorizer

	use(api)
}

func readPrincipal(r *http.Request) {
	// snippet:readPrincipal
	principal := middleware.SecurityPrincipalFrom(r) // any
	scopes := middleware.SecurityScopesFrom(r)       // []string
	route := middleware.MatchedRouteFrom(r)          // *MatchedRoute
	// endsnippet:readPrincipal

	use(principal, scopes, route)
}
