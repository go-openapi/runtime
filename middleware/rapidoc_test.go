package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRapiDocMiddleware(t *testing.T) {
	rapidoc := RapiDoc(RapiDocOpts{}, nil)

	req, _ := http.NewRequest("GET", "/docs", nil)
	recorder := httptest.NewRecorder()
	rapidoc.ServeHTTP(recorder, req)
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "text/html; charset=utf-8", recorder.Header().Get("Content-Type"))
	assert.Contains(t, recorder.Body.String(), "<title>API documentation</title>")
	assert.Contains(t, recorder.Body.String(), "<rapi-doc spec-url=\"/swagger.json\"></rapi-doc>")
	assert.Contains(t, recorder.Body.String(), rapidocLatest)
}
