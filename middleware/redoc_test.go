package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedocMiddleware(t *testing.T) {
	redoc := Redoc(RedocOpts{}, nil)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs", nil)
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	redoc.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "text/html; charset=utf-8", recorder.Header().Get("Content-Type"))
	var o RedocOpts
	o.EnsureDefaults()
	assert.Contains(t, recorder.Body.String(), fmt.Sprintf("<title>%s</title>", o.Title))
	assert.Contains(t, recorder.Body.String(), fmt.Sprintf("<redoc spec-url='%s'></redoc>", o.SpecURL))
	assert.Contains(t, recorder.Body.String(), redocLatest)
}
