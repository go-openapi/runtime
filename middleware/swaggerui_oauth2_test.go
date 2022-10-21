package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSwaggerUIOAuth2CallbackMiddleware(t *testing.T) {
	redoc := SwaggerUIOAuth2Callback(SwaggerUIOpts{}, nil)

	req, _ := http.NewRequest("GET", "/docs/oauth2-callback", nil)
	recorder := httptest.NewRecorder()
	redoc.ServeHTTP(recorder, req)
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "text/html; charset=utf-8", recorder.Header().Get("Content-Type"))
	assert.Contains(t, recorder.Body.String(), "<title>API documentation</title>")
}
