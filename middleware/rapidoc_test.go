package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRapiDocMiddleware(t *testing.T) {
	rapidoc := RapiDoc(RapiDocOpts{}, nil)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs", nil)
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	rapidoc.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "text/html; charset=utf-8", recorder.Header().Get("Content-Type"))
	assert.Contains(t, recorder.Body.String(), "<title>API documentation</title>")
	assert.Contains(t, recorder.Body.String(), "<rapi-doc spec-url=\"/swagger.json\"></rapi-doc>")
	assert.Contains(t, recorder.Body.String(), rapidocLatest)
}
