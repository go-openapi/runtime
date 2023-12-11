package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSwaggerUIMiddleware(t *testing.T) {
	var o SwaggerUIOpts
	o.EnsureDefaults()
	swui := SwaggerUI(o, nil)

	t.Run("with defaults ", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		swui.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)

		assert.Equal(t, "text/html; charset=utf-8", recorder.Header().Get(contentTypeHeader))
		assert.Contains(t, recorder.Body.String(), fmt.Sprintf("<title>%s</title>", o.Title))
		assert.Contains(t, recorder.Body.String(), fmt.Sprintf(`url: '%s',`, strings.ReplaceAll(o.SpecURL, `/`, `\/`)))
		assert.Contains(t, recorder.Body.String(), swaggerLatest)
		assert.Contains(t, recorder.Body.String(), swaggerPresetLatest)
		assert.Contains(t, recorder.Body.String(), swaggerStylesLatest)
		assert.Contains(t, recorder.Body.String(), swaggerFavicon16Latest)
		assert.Contains(t, recorder.Body.String(), swaggerFavicon32Latest)
	})

	t.Run("with path with a trailing / (issue #238)", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs/", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		swui.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("should yield not found", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/nowhere", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		swui.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusNotFound, recorder.Code)
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("with custom template that fails to execute", func(t *testing.T) {
			assert.Panics(t, func() {
				SwaggerUI(SwaggerUIOpts{
					Template: `<!DOCTYPE html>
<html>
	spec-url='{{ .Unknown }}'
</html>
`,
				}, nil)
			})
		})
	})
}
