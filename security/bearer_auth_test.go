package security

import (
	"bytes"
	"context"
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

var bearerAuth = ScopedTokenAuthentication(func(token string, _ []string) (interface{}, error) {
	if token == "token123" {
		return "admin", nil
	}
	return nil, errors.Unauthenticated("bearer")
})

func TestValidBearerAuth(t *testing.T) {
	ba := BearerAuth("owners_auth", bearerAuth)

	req1, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/blah?access_token=token123", nil)
	require.NoError(t, err)

	ok, usr, err := ba.Authenticate(&ScopedAuthRequest{Request: req1})
	assert.True(t, ok)
	assert.Equal(t, "admin", usr)
	require.NoError(t, err)
	assert.Equal(t, "owners_auth", OAuth2SchemeName(req1))

	req2, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/blah", nil)
	require.NoError(t, err)
	req2.Header.Set(runtime.HeaderAuthorization, "Bearer token123")

	ok, usr, err = ba.Authenticate(&ScopedAuthRequest{Request: req2})
	assert.True(t, ok)
	assert.Equal(t, "admin", usr)
	require.NoError(t, err)
	assert.Equal(t, "owners_auth", OAuth2SchemeName(req2))

	body := url.Values(map[string][]string{})
	body.Set("access_token", "token123")
	req3, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/blah", strings.NewReader(body.Encode()))
	require.NoError(t, err)
	req3.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ok, usr, err = ba.Authenticate(&ScopedAuthRequest{Request: req3})
	assert.True(t, ok)
	assert.Equal(t, "admin", usr)
	require.NoError(t, err)
	assert.Equal(t, "owners_auth", OAuth2SchemeName(req3))

	mpbody := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(mpbody)
	err = writer.WriteField("access_token", "token123")
	require.NoError(t, err)
	writer.Close()
	req4, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/blah", mpbody)
	require.NoError(t, err)
	req4.Header.Set("Content-Type", writer.FormDataContentType())

	ok, usr, err = ba.Authenticate(&ScopedAuthRequest{Request: req4})
	assert.True(t, ok)
	assert.Equal(t, "admin", usr)
	require.NoError(t, err)
	assert.Equal(t, "owners_auth", OAuth2SchemeName(req4))
}

//nolint:dupl
func TestInvalidBearerAuth(t *testing.T) {
	ba := BearerAuth("owners_auth", bearerAuth)

	req1, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/blah?access_token=token124", nil)
	require.NoError(t, err)

	ok, usr, err := ba.Authenticate(&ScopedAuthRequest{Request: req1})
	assert.True(t, ok)
	assert.Equal(t, nil, usr)
	require.Error(t, err)

	req2, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/blah", nil)
	require.NoError(t, err)
	req2.Header.Set(runtime.HeaderAuthorization, "Bearer token124")

	ok, usr, err = ba.Authenticate(&ScopedAuthRequest{Request: req2})
	assert.True(t, ok)
	assert.Equal(t, nil, usr)
	require.Error(t, err)

	body := url.Values(map[string][]string{})
	body.Set("access_token", "token124")
	req3, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/blah", strings.NewReader(body.Encode()))
	require.NoError(t, err)
	req3.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ok, usr, err = ba.Authenticate(&ScopedAuthRequest{Request: req3})
	assert.True(t, ok)
	assert.Equal(t, nil, usr)
	require.Error(t, err)

	mpbody := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(mpbody)
	require.NoError(t, writer.WriteField("access_token", "token124"))
	writer.Close()
	req4, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/blah", mpbody)
	require.NoError(t, err)
	req4.Header.Set("Content-Type", writer.FormDataContentType())

	ok, usr, err = ba.Authenticate(&ScopedAuthRequest{Request: req4})
	assert.True(t, ok)
	assert.Equal(t, nil, usr)
	require.Error(t, err)
}

//nolint:dupl
func TestMissingBearerAuth(t *testing.T) {
	ba := BearerAuth("owners_auth", bearerAuth)

	req1, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/blah?access_toke=token123", nil)
	require.NoError(t, err)

	ok, usr, err := ba.Authenticate(&ScopedAuthRequest{Request: req1})
	assert.False(t, ok)
	assert.Equal(t, nil, usr)
	require.NoError(t, err)

	req2, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/blah", nil)
	require.NoError(t, err)
	req2.Header.Set(runtime.HeaderAuthorization, "Beare token123")

	ok, usr, err = ba.Authenticate(&ScopedAuthRequest{Request: req2})
	assert.False(t, ok)
	assert.Equal(t, nil, usr)
	require.NoError(t, err)

	body := url.Values(map[string][]string{})
	body.Set("access_toke", "token123")
	req3, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/blah", strings.NewReader(body.Encode()))
	require.NoError(t, err)
	req3.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ok, usr, err = ba.Authenticate(&ScopedAuthRequest{Request: req3})
	assert.False(t, ok)
	assert.Equal(t, nil, usr)
	require.NoError(t, err)

	mpbody := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(mpbody)
	require.NoError(t, writer.WriteField("access_toke", "token123"))
	writer.Close()
	req4, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/blah", mpbody)
	require.NoError(t, err)
	req4.Header.Set("Content-Type", writer.FormDataContentType())

	ok, usr, err = ba.Authenticate(&ScopedAuthRequest{Request: req4})
	assert.False(t, ok)
	assert.Equal(t, nil, usr)
	require.NoError(t, err)
}

var bearerAuthCtx = ScopedTokenAuthenticationCtx(func(ctx context.Context, token string, requiredScopes []string) (context.Context, interface{}, error) {
	if token == "token123" {
		return context.WithValue(ctx, extra, extraWisdom), "admin", nil
	}
	return context.WithValue(ctx, reason, expReason), nil, errors.Unauthenticated("bearer")
})

func TestValidBearerAuthCtx(t *testing.T) {
	ba := BearerAuthCtx("owners_auth", bearerAuthCtx)

	req1, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/blah?access_token=token123", nil)
	require.NoError(t, err)
	req1 = req1.WithContext(context.WithValue(req1.Context(), original, wisdom))
	ok, usr, err := ba.Authenticate(&ScopedAuthRequest{Request: req1})
	assert.True(t, ok)
	assert.Equal(t, "admin", usr)
	require.NoError(t, err)
	assert.Equal(t, wisdom, req1.Context().Value(original))
	assert.Equal(t, extraWisdom, req1.Context().Value(extra))
	assert.Nil(t, req1.Context().Value(reason))
	assert.Equal(t, "owners_auth", OAuth2SchemeName(req1))

	req2, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/blah", nil)
	require.NoError(t, err)
	req2 = req2.WithContext(context.WithValue(req2.Context(), original, wisdom))
	req2.Header.Set(runtime.HeaderAuthorization, "Bearer token123")

	ok, usr, err = ba.Authenticate(&ScopedAuthRequest{Request: req2})
	assert.True(t, ok)
	assert.Equal(t, "admin", usr)
	require.NoError(t, err)
	assert.Equal(t, wisdom, req2.Context().Value(original))
	assert.Equal(t, extraWisdom, req2.Context().Value(extra))
	assert.Nil(t, req2.Context().Value(reason))
	assert.Equal(t, "owners_auth", OAuth2SchemeName(req2))

	body := url.Values(map[string][]string{})
	body.Set("access_token", "token123")
	req3, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/blah", strings.NewReader(body.Encode()))
	require.NoError(t, err)
	req3 = req3.WithContext(context.WithValue(req3.Context(), original, wisdom))
	req3.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ok, usr, err = ba.Authenticate(&ScopedAuthRequest{Request: req3})
	assert.True(t, ok)
	assert.Equal(t, "admin", usr)
	require.NoError(t, err)
	assert.Equal(t, wisdom, req3.Context().Value(original))
	assert.Equal(t, extraWisdom, req3.Context().Value(extra))
	assert.Nil(t, req3.Context().Value(reason))
	assert.Equal(t, "owners_auth", OAuth2SchemeName(req3))

	mpbody := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(mpbody)
	require.NoError(t, writer.WriteField("access_token", "token123"))
	writer.Close()
	req4, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/blah", mpbody)
	require.NoError(t, err)
	req4 = req4.WithContext(context.WithValue(req4.Context(), original, wisdom))
	req4.Header.Set("Content-Type", writer.FormDataContentType())

	ok, usr, err = ba.Authenticate(&ScopedAuthRequest{Request: req4})
	assert.True(t, ok)
	assert.Equal(t, "admin", usr)
	require.NoError(t, err)
	assert.Equal(t, wisdom, req4.Context().Value(original))
	assert.Equal(t, extraWisdom, req4.Context().Value(extra))
	assert.Nil(t, req4.Context().Value(reason))
	assert.Equal(t, "owners_auth", OAuth2SchemeName(req4))
}

func TestInvalidBearerAuthCtx(t *testing.T) {
	ba := BearerAuthCtx("owners_auth", bearerAuthCtx)

	req1, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/blah?access_token=token124", nil)
	require.NoError(t, err)
	req1 = req1.WithContext(context.WithValue(req1.Context(), original, wisdom))
	ok, usr, err := ba.Authenticate(&ScopedAuthRequest{Request: req1})
	assert.True(t, ok)
	assert.Equal(t, nil, usr)
	require.Error(t, err)
	assert.Equal(t, wisdom, req1.Context().Value(original))
	assert.Equal(t, expReason, req1.Context().Value(reason))
	assert.Nil(t, req1.Context().Value(extra))

	req2, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/blah", nil)
	require.NoError(t, err)
	req2 = req2.WithContext(context.WithValue(req2.Context(), original, wisdom))
	req2.Header.Set(runtime.HeaderAuthorization, "Bearer token124")

	ok, usr, err = ba.Authenticate(&ScopedAuthRequest{Request: req2})
	assert.True(t, ok)
	assert.Equal(t, nil, usr)
	require.Error(t, err)
	assert.Equal(t, wisdom, req2.Context().Value(original))
	assert.Equal(t, expReason, req2.Context().Value(reason))
	assert.Nil(t, req2.Context().Value(extra))

	body := url.Values(map[string][]string{})
	body.Set("access_token", "token124")
	req3, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/blah", strings.NewReader(body.Encode()))
	require.NoError(t, err)
	req3 = req3.WithContext(context.WithValue(req3.Context(), original, wisdom))
	req3.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ok, usr, err = ba.Authenticate(&ScopedAuthRequest{Request: req3})
	assert.True(t, ok)
	assert.Equal(t, nil, usr)
	require.Error(t, err)
	assert.Equal(t, wisdom, req3.Context().Value(original))
	assert.Equal(t, expReason, req3.Context().Value(reason))
	assert.Nil(t, req3.Context().Value(extra))

	mpbody := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(mpbody)
	require.NoError(t, writer.WriteField("access_token", "token124"))
	writer.Close()
	req4, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/blah", mpbody)
	require.NoError(t, err)
	req4 = req4.WithContext(context.WithValue(req4.Context(), original, wisdom))
	req4.Header.Set("Content-Type", writer.FormDataContentType())

	ok, usr, err = ba.Authenticate(&ScopedAuthRequest{Request: req4})
	assert.True(t, ok)
	assert.Equal(t, nil, usr)
	require.Error(t, err)
	assert.Equal(t, wisdom, req4.Context().Value(original))
	assert.Equal(t, expReason, req4.Context().Value(reason))
	assert.Nil(t, req4.Context().Value(extra))
}

func TestMissingBearerAuthCtx(t *testing.T) {
	ba := BearerAuthCtx("owners_auth", bearerAuthCtx)

	req1, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/blah?access_toke=token123", nil)
	require.NoError(t, err)
	req1 = req1.WithContext(context.WithValue(req1.Context(), original, wisdom))
	ok, usr, err := ba.Authenticate(&ScopedAuthRequest{Request: req1})
	assert.False(t, ok)
	assert.Equal(t, nil, usr)
	require.NoError(t, err)
	assert.Equal(t, wisdom, req1.Context().Value(original))
	assert.Nil(t, req1.Context().Value(reason))
	assert.Nil(t, req1.Context().Value(extra))

	req2, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/blah", nil)
	require.NoError(t, err)
	req2.Header.Set(runtime.HeaderAuthorization, "Beare token123")

	ok, usr, err = ba.Authenticate(&ScopedAuthRequest{Request: req2})
	req2 = req2.WithContext(context.WithValue(req2.Context(), original, wisdom))
	assert.False(t, ok)
	assert.Equal(t, nil, usr)
	require.NoError(t, err)
	assert.Equal(t, wisdom, req2.Context().Value(original))
	assert.Nil(t, req2.Context().Value(reason))
	assert.Nil(t, req2.Context().Value(extra))

	body := url.Values(map[string][]string{})
	body.Set("access_toke", "token123")
	req3, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/blah", strings.NewReader(body.Encode()))
	require.NoError(t, err)
	req3 = req3.WithContext(context.WithValue(req3.Context(), original, wisdom))
	req3.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ok, usr, err = ba.Authenticate(&ScopedAuthRequest{Request: req3})
	assert.False(t, ok)
	assert.Equal(t, nil, usr)
	require.NoError(t, err)
	assert.Equal(t, wisdom, req3.Context().Value(original))
	assert.Nil(t, req3.Context().Value(reason))
	assert.Nil(t, req3.Context().Value(extra))

	mpbody := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(mpbody)
	err = writer.WriteField("access_toke", "token123")
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	req4, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/blah", mpbody)
	require.NoError(t, err)
	req4 = req4.WithContext(context.WithValue(req4.Context(), original, wisdom))
	req4.Header.Set("Content-Type", writer.FormDataContentType())

	ok, usr, err = ba.Authenticate(&ScopedAuthRequest{Request: req4})
	assert.False(t, ok)
	assert.Equal(t, nil, usr)
	require.NoError(t, err)
	assert.Equal(t, wisdom, req4.Context().Value(original))
	assert.Nil(t, req4.Context().Value(reason))
	assert.Nil(t, req4.Context().Value(extra))
}
