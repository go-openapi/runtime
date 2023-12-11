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
	t.Run("with defaults", func(t *testing.T) {
		redoc := Redoc(RedocOpts{}, nil)

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()
		redoc.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "text/html; charset=utf-8", recorder.Header().Get(contentTypeHeader))
		var o RedocOpts
		o.EnsureDefaults()
		assert.Contains(t, recorder.Body.String(), fmt.Sprintf("<title>%s</title>", o.Title))
		assert.Contains(t, recorder.Body.String(), fmt.Sprintf("<redoc spec-url='%s'></redoc>", o.SpecURL))
		assert.Contains(t, recorder.Body.String(), redocLatest)
	})

	t.Run("with alternate path and spec URL", func(t *testing.T) {
		redoc := Redoc(RedocOpts{
			BasePath: "/base",
			Path:     "ui",
			SpecURL:  "/ui/swagger.json",
		}, nil)

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/base/ui", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()
		redoc.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "<redoc spec-url='/ui/swagger.json'></redoc>")
	})

	t.Run("with custom template", func(t *testing.T) {
		redoc := Redoc(RedocOpts{
			Template: `<!DOCTYPE html>
<html>
  <head>
    <title>{{ .Title }}</title>
		<meta charset="utf-8"/>
		<meta name="viewport" content="width=device-width, initial-scale=1">
		<link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">

    <!--
    ReDoc doesn't change outer page styles
    -->
    <style>
      body {
        margin: 0;
        padding: 0;
      }
    </style>
  </head>
  <body>
    <redoc
				spec-url='{{ .SpecURL }}'
				required-props-first=true
        theme='{
         "sidebar": {
           "backgroundColor": "lightblue"
         }
        }'
		></redoc>
    <script src="{{ .RedocURL }}"> </script>
  </body>
</html>
`,
		}, nil)

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()
		redoc.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "required-props-first=true")
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("with invalid custom template", func(t *testing.T) {
			assert.Panics(t, func() {
				Redoc(RedocOpts{
					Template: `<!DOCTYPE html>
<html>
  <head>
				spec-url='{{ .Spec
</html>
`,
				}, nil)
			})
		})

		t.Run("with custom template that fails to execute", func(t *testing.T) {
			assert.Panics(t, func() {
				Redoc(RedocOpts{
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
