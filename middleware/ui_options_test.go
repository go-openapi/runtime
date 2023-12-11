package middleware

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConvertOptions(t *testing.T) {
	t.Run("from any UI options to uiOptions", func(t *testing.T) {
		t.Run("from RedocOpts", func(t *testing.T) {
			in := RedocOpts{
				BasePath: "a",
				Path:     "b",
				SpecURL:  "c",
				Template: "d",
				Title:    "e",
				RedocURL: "f",
			}
			out := toCommonUIOptions(in)

			require.Equal(t, "a", out.BasePath)
			require.Equal(t, "b", out.Path)
			require.Equal(t, "c", out.SpecURL)
			require.Equal(t, "d", out.Template)
			require.Equal(t, "e", out.Title)
		})

		t.Run("from RapiDocOpts", func(t *testing.T) {
			in := RapiDocOpts{
				BasePath:   "a",
				Path:       "b",
				SpecURL:    "c",
				Template:   "d",
				Title:      "e",
				RapiDocURL: "f",
			}
			out := toCommonUIOptions(in)

			require.Equal(t, "a", out.BasePath)
			require.Equal(t, "b", out.Path)
			require.Equal(t, "c", out.SpecURL)
			require.Equal(t, "d", out.Template)
			require.Equal(t, "e", out.Title)
		})

		t.Run("from SwaggerUIOpts", func(t *testing.T) {
			in := SwaggerUIOpts{
				BasePath:   "a",
				Path:       "b",
				SpecURL:    "c",
				Template:   "d",
				Title:      "e",
				SwaggerURL: "f",
			}
			out := toCommonUIOptions(in)

			require.Equal(t, "a", out.BasePath)
			require.Equal(t, "b", out.Path)
			require.Equal(t, "c", out.SpecURL)
			require.Equal(t, "d", out.Template)
			require.Equal(t, "e", out.Title)
		})
	})

	t.Run("from uiOptions to any UI options", func(t *testing.T) {
		in := uiOptions{
			BasePath: "a",
			Path:     "b",
			SpecURL:  "c",
			Template: "d",
			Title:    "e",
		}

		t.Run("to RedocOpts", func(t *testing.T) {
			var out RedocOpts
			fromCommonToAnyOptions(in, &out)
			require.Equal(t, "a", out.BasePath)
			require.Equal(t, "b", out.Path)
			require.Equal(t, "c", out.SpecURL)
			require.Equal(t, "d", out.Template)
			require.Equal(t, "e", out.Title)
		})

		t.Run("to RapiDocOpts", func(t *testing.T) {
			var out RapiDocOpts
			fromCommonToAnyOptions(in, &out)
			require.Equal(t, "a", out.BasePath)
			require.Equal(t, "b", out.Path)
			require.Equal(t, "c", out.SpecURL)
			require.Equal(t, "d", out.Template)
			require.Equal(t, "e", out.Title)
		})

		t.Run("to SwaggerUIOpts", func(t *testing.T) {
			var out SwaggerUIOpts
			fromCommonToAnyOptions(in, &out)
			require.Equal(t, "a", out.BasePath)
			require.Equal(t, "b", out.Path)
			require.Equal(t, "c", out.SpecURL)
			require.Equal(t, "d", out.Template)
			require.Equal(t, "e", out.Title)
		})
	})
}
