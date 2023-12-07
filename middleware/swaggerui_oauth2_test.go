package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSwaggerUIOAuth2CallbackMiddleware(t *testing.T) {
	redoc := SwaggerUIOAuth2Callback(SwaggerUIOpts{}, nil)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs/oauth2-callback", nil)
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	redoc.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "text/html; charset=utf-8", recorder.Header().Get("Content-Type"))
	assert.Contains(t, recorder.Body.String(), "<title>API documentation</title>")
}