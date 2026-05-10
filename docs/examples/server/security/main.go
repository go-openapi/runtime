// SPDX-License-Identifier: Apache-2.0

// Command security backs the snippets on the doc-site
// "Security schemes" page. Each function below is the source of a
// `{{< code region="..." >}}` include; the package as a whole compiles
// and lints so the snippets cannot rot silently.
//
// `go run .` exercises the non-blocking demos (wiring registration).
package main

import (
	"context"
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/runtime/middleware/untyped"
	"github.com/go-openapi/runtime/security"
)

// --- Stubs (excluded from rendered snippets) ------------------------

// doc is a placeholder OpenAPI document. Snippets pretend it was loaded
// from disk; the demo wires a freshly-constructed empty spec so the
// program compiles and runs.
var doc *loads.Document

// appPrincipal stands in for whatever the application returns from
// authentication (e.g. *models.Principal).
type appPrincipal struct {
	ID     string
	Email  string
	scopes []string
}

// HasScopes reports whether the principal carries every required scope.
func (p *appPrincipal) HasScopes(required []string) bool {
	have := make(map[string]struct{}, len(p.scopes))
	for _, s := range p.scopes {
		have[s] = struct{}{}
	}
	for _, s := range required {
		if _, ok := have[s]; !ok {
			return false
		}
	}
	return true
}

// fakeStore stands in for an application-supplied user / token store.
type fakeStore struct{}

func (fakeStore) AuthenticateBasic(_ context.Context, user, _ string) (*appPrincipal, error) {
	if user == "" {
		return nil, errors.Unauthenticated("basic")
	}
	return &appPrincipal{ID: user, Email: user + "@example.com"}, nil
}

func (fakeStore) AuthenticateAPIKey(token string) (*appPrincipal, error) {
	if token == "" {
		return nil, errors.Unauthenticated("api-key")
	}
	return &appPrincipal{ID: token}, nil
}

var store = fakeStore{}

// fakeTokens stands in for a JWT / opaque-token verifier.
type fakeTokens struct{}

func (fakeTokens) Verify(token string) (*appPrincipal, bool) {
	if token == "" {
		return nil, false
	}
	return &appPrincipal{ID: token, scopes: []string{"read:pets"}}, true
}

var tokens = fakeTokens{}

// auditPkg stands in for an application-supplied audit / tracing
// context helper — the doc snippet calls `audit.WithUser(ctx, id)`.
type auditPkg struct{}

type auditKey struct{}

func (auditPkg) WithUser(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, auditKey{}, id)
}

var audit = auditPkg{}

// use silences "declared and not used" diagnostics for snippet-local
// values that exist only to make the demo compile.
func use(_ ...any) {}

func main() {
	basicAuthCtx()
	basicAuthSimple()
	apiKeyAuthHeader()
	bearerAuthScopes()
	registerAuthorized()
	readPrincipal(nil)
}

// --- Snippets -------------------------------------------------------

func basicAuthCtx() {
	// snippet:basicAuthCtx
	authn := security.BasicAuthCtx(func(ctx context.Context, user, pass string) (context.Context, any, error) {
		// request-scoped DB call honours ctx cancellation
		principal, err := store.AuthenticateBasic(ctx, user, pass)
		if err != nil {
			return ctx, nil, err
		}
		// enrich the context for downstream handlers
		ctx = audit.WithUser(ctx, principal.ID)
		return ctx, principal, nil
	})
	// endsnippet:basicAuthCtx

	use(authn)
}

func basicAuthSimple() {
	// snippet:basicAuthSimple
	// principal type is up to you
	type Principal struct {
		ID    string
		Email string
	}

	authn := security.BasicAuth(func(user, _ string) (any, error) {
		if user == "" {
			return nil, errors.Unauthenticated("basic")
		}
		return Principal{ID: user, Email: user + "@example.com"}, nil
	})
	// endsnippet:basicAuthSimple

	use(authn)
}

func apiKeyAuthHeader() {
	// snippet:apiKeyAuthHeader
	authn := security.APIKeyAuth("X-Api-Key", "header",
		func(token string) (any, error) {
			return store.AuthenticateAPIKey(token)
		},
	)
	// endsnippet:apiKeyAuthHeader

	use(authn)
}

func bearerAuthScopes() {
	// snippet:bearerAuthScopes
	authn := security.BearerAuth("oauth2",
		func(token string, requiredScopes []string) (any, error) {
			principal, ok := tokens.Verify(token)
			if !ok {
				return nil, errors.Unauthenticated("bearer")
			}
			if !principal.HasScopes(requiredScopes) {
				return nil, errors.New(http.StatusForbidden, "insufficient_scope")
			}
			return principal, nil
		},
	)
	// endsnippet:bearerAuthScopes

	use(authn)
}

func registerAuthorized() {
	api := untyped.NewAPI(doc).WithJSONDefaults()

	// snippet:registerAuthorized
	api.RegisterAuthorizer(security.Authorized()) // always allow
	// endsnippet:registerAuthorized

	use(api)
}

func readPrincipal(r *http.Request) {
	if r == nil {
		// readPrincipal is invoked from main() for compile coverage; the
		// snippet body itself is what gets rendered into the docs.
		return
	}

	// snippet:readPrincipal
	principal := middleware.SecurityPrincipalFrom(r)
	scopes := middleware.SecurityScopesFrom(r)
	// endsnippet:readPrincipal

	use(principal, scopes)
}
