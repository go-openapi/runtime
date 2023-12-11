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

func TestRapiDocMiddleware(t *testing.T) {
	t.Run("with defaults", func(t *testing.T) {
		rapidoc := RapiDoc(RapiDocOpts{}, nil)

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()
		rapidoc.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "text/html; charset=utf-8", recorder.Header().Get(contentTypeHeader))
		var o RapiDocOpts
		o.EnsureDefaults()
		assert.Contains(t, recorder.Body.String(), fmt.Sprintf("<title>%s</title>", o.Title))
		assert.Contains(t, recorder.Body.String(), fmt.Sprintf("<rapi-doc spec-url=%q></rapi-doc>", o.SpecURL))
		assert.Contains(t, recorder.Body.String(), rapidocLatest)
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("with custom template that fails to execute", func(t *testing.T) {
			assert.Panics(t, func() {
				RapiDoc(RapiDocOpts{
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
