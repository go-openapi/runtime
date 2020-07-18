package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSwaggerUIMiddleware(t *testing.T) {
	redoc := SwaggerUI(SwaggerUIOpts{}, nil)

	req, _ := http.NewRequest("GET", "/docs", nil)
	recorder := httptest.NewRecorder()
	redoc.ServeHTTP(recorder, req)
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "text/html; charset=utf-8", recorder.Header().Get("Content-Type"))
	assert.Contains(t, recorder.Body.String(), "<title>API documentation</title>")
	assert.Contains(t, recorder.Body.String(), "url: '\\/swagger.json',")
	assert.Contains(t, recorder.Body.String(), swaggerLatest)
	assert.Contains(t, recorder.Body.String(), swaggerPresetLatest)
	assert.Contains(t, recorder.Body.String(), swaggerStylesLatest)
	assert.Contains(t, recorder.Body.String(), swaggerFavicon16Latest)
	assert.Contains(t, recorder.Body.String(), swaggerFavicon32Latest)
}
