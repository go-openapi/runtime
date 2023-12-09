package security

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	owners       = "owners_auth"
	validToken   = "token123"
	invalidToken = "token124"
	principal    = "admin"
	authPath     = "/blah"
	invalidParam = "access_toke"
)

type authExpectation uint8

const (
	expectIsAuthorized authExpectation = iota
	expectInvalidAuthorization
	expectNoAuthorization
)

func TestBearerAuth(t *testing.T) {
	bearerAuth := ScopedTokenAuthentication(func(token string, _ []string) (interface{}, error) {
		if token == validToken {
			return principal, nil
		}
		return nil, errors.Unauthenticated("bearer")
	})
	ba := BearerAuth(owners, bearerAuth)
	ctx := context.Background()

	t.Run("with valid bearer auth", func(t *testing.T) {
		t.Run("token in query param",
			testAuthenticateBearerInQuery(ctx, ba, "", validToken, expectIsAuthorized),
		)
		t.Run("token in header",
			testAuthenticateBearerInHeader(ctx, ba, "", validToken, expectIsAuthorized),
		)
		t.Run("token in urlencoded form",
			testAuthenticateBearerInForm(ctx, ba, "", validToken, expectIsAuthorized),
		)
		t.Run("token in multipart form",
			testAuthenticateBearerInMultipartForm(ctx, ba, "", validToken, expectIsAuthorized),
		)
	})

	t.Run("with invalid token", func(t *testing.T) {
		t.Run("token in query param",
			testAuthenticateBearerInQuery(ctx, ba, "", invalidToken, expectInvalidAuthorization),
		)
		t.Run("token in header",
			testAuthenticateBearerInHeader(ctx, ba, "", invalidToken, expectInvalidAuthorization),
		)
		t.Run("token in urlencoded form",
			testAuthenticateBearerInForm(ctx, ba, "", invalidToken, expectInvalidAuthorization),
		)
		t.Run("token in multipart form",
			testAuthenticateBearerInMultipartForm(ctx, ba, "", invalidToken, expectInvalidAuthorization),
		)
	})

	t.Run("with missing auth", func(t *testing.T) {
		t.Run("token in query param",
			testAuthenticateBearerInQuery(ctx, ba, invalidParam, validToken, expectNoAuthorization),
		)
		t.Run("token in header",
			testAuthenticateBearerInHeader(ctx, ba, "Beare", validToken, expectNoAuthorization),
		)
		t.Run("token in urlencoded form",
			testAuthenticateBearerInForm(ctx, ba, invalidParam, validToken, expectNoAuthorization),
		)
		t.Run("token in multipart form",
			testAuthenticateBearerInMultipartForm(ctx, ba, invalidParam, validToken, expectNoAuthorization),
		)
	})
}

func TestBearerAuthCtx(t *testing.T) {
	bearerAuthCtx := ScopedTokenAuthenticationCtx(func(ctx context.Context, token string, _ []string) (context.Context, interface{}, error) {
		if token == validToken {
			return context.WithValue(ctx, extra, extraWisdom), principal, nil
		}
		return context.WithValue(ctx, reason, expReason), nil, errors.Unauthenticated("bearer")
	})
	ba := BearerAuthCtx(owners, bearerAuthCtx)
	ctx := context.WithValue(context.Background(), original, wisdom)

	assertContextOK := func(requestContext context.Context, t *testing.T) {
		// when authorized, we have an "extra" key in context
		assert.Equal(t, wisdom, requestContext.Value(original))
		assert.Equal(t, extraWisdom, requestContext.Value(extra))
		assert.Nil(t, requestContext.Value(reason))
	}

	assertContextKO := func(requestContext context.Context, t *testing.T) {
		// when not authorized, we have a "reason" key in context
		assert.Equal(t, wisdom, requestContext.Value(original))
		assert.Nil(t, requestContext.Value(extra))
		assert.Equal(t, expReason, requestContext.Value(reason))
	}

	assertContextNone := func(requestContext context.Context, t *testing.T) {
		// when missing authorization, we only have the original context
		assert.Equal(t, wisdom, requestContext.Value(original))
		assert.Nil(t, requestContext.Value(extra))
		assert.Nil(t, requestContext.Value(reason))
	}

	t.Run("with valid bearer auth", func(t *testing.T) {
		t.Run("token in query param",
			testAuthenticateBearerInQuery(ctx, ba, "", validToken, expectIsAuthorized, assertContextOK),
		)
		t.Run("token in header",
			testAuthenticateBearerInHeader(ctx, ba, "", validToken, expectIsAuthorized, assertContextOK),
		)
		t.Run("token in urlencoded form",
			testAuthenticateBearerInForm(ctx, ba, "", validToken, expectIsAuthorized, assertContextOK),
		)
		t.Run("token in multipart form",
			testAuthenticateBearerInMultipartForm(ctx, ba, "", validToken, expectIsAuthorized, assertContextOK),
		)
	})

	t.Run("with invalid token", func(t *testing.T) {
		t.Run("token in query param",
			testAuthenticateBearerInQuery(ctx, ba, "", invalidToken, expectInvalidAuthorization, assertContextKO),
		)
		t.Run("token in header",
			testAuthenticateBearerInHeader(ctx, ba, "", invalidToken, expectInvalidAuthorization, assertContextKO),
		)
		t.Run("token in urlencoded form",
			testAuthenticateBearerInForm(ctx, ba, "", invalidToken, expectInvalidAuthorization, assertContextKO),
		)
		t.Run("token in multipart form",
			testAuthenticateBearerInMultipartForm(ctx, ba, "", invalidToken, expectInvalidAuthorization, assertContextKO),
		)
	})

	t.Run("with missing auth", func(t *testing.T) {
		t.Run("token in query param",
			testAuthenticateBearerInQuery(ctx, ba, invalidParam, validToken, expectNoAuthorization, assertContextNone),
		)
		t.Run("token in header",
			testAuthenticateBearerInHeader(ctx, ba, "Beare", validToken, expectNoAuthorization, assertContextNone),
		)
		t.Run("token in urlencoded form",
			testAuthenticateBearerInForm(ctx, ba, invalidParam, validToken, expectNoAuthorization, assertContextNone),
		)
		t.Run("token in multipart form",
			testAuthenticateBearerInMultipartForm(ctx, ba, invalidParam, validToken, expectNoAuthorization, assertContextNone),
		)
	})
}

func testIsAuthorized(_ context.Context, req *http.Request, authorizer runtime.Authenticator, expectAuthorized authExpectation, extraAsserters ...func(context.Context, *testing.T)) func(*testing.T) {
	return func(t *testing.T) {
		hasToken, usr, err := authorizer.Authenticate(&ScopedAuthRequest{Request: req})
		switch expectAuthorized {

		case expectIsAuthorized:
			require.NoError(t, err)
			assert.True(t, hasToken)
			assert.Equal(t, principal, usr)
			assert.Equal(t, owners, OAuth2SchemeName(req))

		case expectInvalidAuthorization:
			require.Error(t, err)
			require.ErrorContains(t, err, "unauthenticated")
			assert.True(t, hasToken)
			assert.Nil(t, usr)
			assert.Equal(t, owners, OAuth2SchemeName(req))

		case expectNoAuthorization:
			require.NoError(t, err)
			assert.False(t, hasToken)
			assert.Nil(t, usr)
			assert.Empty(t, OAuth2SchemeName(req))
		}

		for _, contextAsserter := range extraAsserters {
			contextAsserter(req.Context(), t)
		}
	}
}

func shouldAuthorizeOrNot(expectAuthorized authExpectation) string {
	if expectAuthorized == expectIsAuthorized {
		return "should authorize"
	}

	return "should not authorize"
}

func testAuthenticateBearerInQuery(
	// build a request with the token as a query parameter, then check against the authorizer
	//
	// the request context after authorization may be checked with the extraAsserters.
	ctx context.Context, authorizer runtime.Authenticator, parameter, token string, expectAuthorized authExpectation,
	extraAsserters ...func(context.Context, *testing.T),
) func(*testing.T) {
	if parameter == "" {
		parameter = accessTokenParam
	}

	return func(t *testing.T) {
		req, err := http.NewRequestWithContext(
			ctx, http.MethodGet,
			fmt.Sprintf("%s?%s=%s", authPath, parameter, token),
			nil,
		)
		require.NoError(t, err)

		t.Run(
			shouldAuthorizeOrNot(expectAuthorized),
			testIsAuthorized(ctx, req, authorizer, expectAuthorized, extraAsserters...),
		)
	}
}

func testAuthenticateBearerInHeader(
	// build a request with the token as a header, then check against the authorizer
	ctx context.Context, authorizer runtime.Authenticator, parameter, token string, expectAuthorized authExpectation,
	extraAsserters ...func(context.Context, *testing.T),
) func(*testing.T) {
	if parameter == "" {
		parameter = "Bearer"
	}

	return func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, authPath, nil)
		require.NoError(t, err)
		req.Header.Set(runtime.HeaderAuthorization, fmt.Sprintf("%s %s", parameter, token))

		t.Run(
			shouldAuthorizeOrNot(expectAuthorized),
			testIsAuthorized(ctx, req, authorizer, expectAuthorized, extraAsserters...),
		)
	}
}

func testAuthenticateBearerInForm(
	// build a request with the token as a form field, then check against the authorizer
	ctx context.Context, authorizer runtime.Authenticator, parameter, token string, expectAuthorized authExpectation,
	extraAsserters ...func(context.Context, *testing.T),
) func(*testing.T) {
	if parameter == "" {
		parameter = accessTokenParam
	}

	return func(t *testing.T) {
		body := url.Values(map[string][]string{})
		body.Set(parameter, token)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, authPath, strings.NewReader(body.Encode()))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		t.Run(
			shouldAuthorizeOrNot(expectAuthorized),
			testIsAuthorized(ctx, req, authorizer, expectAuthorized, extraAsserters...),
		)
	}
}
func testAuthenticateBearerInMultipartForm(
	// build a request with the token as a multipart form field, then check against the authorizer
	ctx context.Context, authorizer runtime.Authenticator, parameter, token string, expectAuthorized authExpectation,
	extraAsserters ...func(context.Context, *testing.T),
) func(*testing.T) {
	if parameter == "" {
		parameter = accessTokenParam
	}

	return func(t *testing.T) {
		body := bytes.NewBuffer(nil)
		writer := multipart.NewWriter(body)
		require.NoError(t, writer.WriteField(parameter, token))
		require.NoError(t, writer.Close())
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, authPath, body)
		require.NoError(t, err)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		t.Run(
			shouldAuthorizeOrNot(expectAuthorized),
			testIsAuthorized(ctx, req, authorizer, expectAuthorized, extraAsserters...),
		)
	}
}
