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

func TestSwaggerUIOAuth2CallbackMiddleware(t *testing.T) {
	t.Run("with defaults", func(t *testing.T) {
		doc := SwaggerUIOAuth2Callback(SwaggerUIOpts{}, nil)

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs/oauth2-callback", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		doc.ServeHTTP(recorder, req)
		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "text/html; charset=utf-8", recorder.Header().Get(contentTypeHeader))

		var o SwaggerUIOpts
		o.EnsureDefaultsOauth2()
		htmlResponse := recorder.Body.String()
		assert.Contains(t, htmlResponse, fmt.Sprintf("<title>%s</title>", o.Title))
		assert.Contains(t, htmlResponse, `oauth2.auth.schema.get("flow") === "accessCode"`)
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("with custom template that fails to execute", func(t *testing.T) {
			assert.Panics(t, func() {
				SwaggerUIOAuth2Callback(SwaggerUIOpts{
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
