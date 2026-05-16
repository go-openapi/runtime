// SPDX-License-Identifier: Apache-2.0

// Command oauth2 backs the snippets on the doc-site
// "OAuth2 access-code (Google)" recipe page
// (usage/examples/auth/oauth2-access-code.md).
//
// The runtime only enters the OAuth2 access-code dance on the
// protected-route side: a Bearer authenticator validates the inbound
// access token via an introspection helper. The /login and
// /auth/callback handlers are plain redirect/exchange handlers that
// live in user code.
//
// External dependencies on `golang.org/x/oauth2` and `coreos/go-oidc`
// are deliberately not imported here — they are illustrative of *user*
// wiring, not of the runtime API. The OAuth2 client and userinfo
// validator are stubbed at package level so the snippet itself shows
// real runtime calls without bloating the doc-examples module with
// auth provider SDKs.
//
// `go run .` exercises the demo wiring against a no-op spec.
package main

import (
	"context"
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/runtime/middleware/untyped"
	"github.com/go-openapi/runtime/security"
)

// --- Stubs (excluded from rendered snippets) ------------------------

// doc stands in for a real `*loads.Document` loaded via `loads.Spec`.
var doc *loads.Document

// state is the single-shot CSRF token shown for brevity in the
// recipe. In production this MUST be a per-session unguessable value.
var state = "foobar"

// oauth2Token mirrors the shape of `*oauth2.Token` used by the snippet
// without pulling `golang.org/x/oauth2` into the doc-examples module.
type oauth2Token struct {
	AccessToken string
}

// oauth2Config mirrors the slice of `oauth2.Config` the snippet calls
// into: `AuthCodeURL(state)` for the redirect and `Exchange(ctx, code)`
// for the token swap. The real type lives in `golang.org/x/oauth2`.
type oauth2Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

func (oauth2Config) AuthCodeURL(_ string) string { return "" }

func (oauth2Config) Exchange(_ context.Context, _ string) (*oauth2Token, error) {
	return &oauth2Token{}, nil
}

// config is the application's OAuth2 client config. See the inline
// "Application configuration" block in the recipe for the real
// `oauth2.Config` literal — it is left fenced (not migrated) because
// it is pure external-library wiring.
var config oauth2Config

// validateAtUserInfoURL is a stand-in for the plain HTTP call to the
// provider's userinfo endpoint with the bearer token. Replace with a
// real userinfo fetch in production code.
func validateAtUserInfoURL(_ string) (bool, error) { return true, nil }

// currentRequest stands in for the inbound *http.Request that
// `middleware.ResponderFunc` does not expose. In real generated code
// the handler receives bound params (with `HTTPRequest`); the recipe
// inlines the redirect call to keep the snippet short.
var currentRequest *http.Request

// getCallbackParams mirrors the bound params struct go-swagger
// generates for the `/auth/callback` operation. The untyped flow
// exposes the raw HTTP request through the same `HTTPRequest` field.
type getCallbackParams struct {
	HTTPRequest *http.Request
}

// use silences "declared and not used" diagnostics for snippet-local
// values that exist only to make the demo compile.
func use(_ ...any) {}

func main() {
	wireOauth2AccessCode()
}

// --- Snippets -------------------------------------------------------

func wireOauth2AccessCode() {
	// snippet:wireOauth2AccessCode
	api := untyped.NewAPI(doc).WithJSONDefaults()

	// 1. The protected-route validator. Called for any operation whose
	//    `security:` block lists `OauthSecurity`.
	api.RegisterAuth("OauthSecurity", security.BearerAuth("OauthSecurity",
		func(token string, _ []string) (any, error) {
			ok, err := validateAtUserInfoURL(token)
			if err != nil || !ok {
				return nil, errors.Unauthenticated("bearer")
			}
			return token, nil
		},
	))

	// 2. /login redirects the browser to Google's auth endpoint.
	api.RegisterOperation("get", "/login", runtime.OperationHandlerFunc(
		func(_ any) (any, error) {
			return middleware.ResponderFunc(func(w http.ResponseWriter, _ runtime.Producer) {
				// params.HTTPRequest is unused here — pass nil since middleware.Redirect ignores it
				http.Redirect(w, currentRequest, config.AuthCodeURL(state), http.StatusFound)
			}), nil
		},
	))

	// 3. /auth/callback exchanges the code Google returns for an access token.
	api.RegisterOperation("get", "/auth/callback", runtime.OperationHandlerFunc(
		func(params any) (any, error) {
			// The bound params struct exposes the raw HTTP request through HTTPRequest.
			// Untyped: extract from params; typed: it's already a field.
			callbackParams, ok := params.(getCallbackParams)
			if !ok {
				panic("internal error")
			}
			r := callbackParams.HTTPRequest
			if r.URL.Query().Get("state") != state {
				return nil, errors.New(http.StatusBadRequest, "state mismatch")
			}
			token, err := config.Exchange(r.Context(), r.URL.Query().Get("code"))
			if err != nil {
				return nil, errors.New(http.StatusInternalServerError, "token exchange failed")
			}
			return map[string]string{"access_token": token.AccessToken}, nil
		},
	))
	// endsnippet:wireOauth2AccessCode

	use(api)
}
