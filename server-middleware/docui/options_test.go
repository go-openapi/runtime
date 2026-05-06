// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package docui

import (
	"testing"

	"github.com/go-openapi/testify/v2/require"
)

func TestConvertOptions(t *testing.T) {
	t.Run("from any UI options to UIOptions", func(t *testing.T) {
		t.Run("from RedocOpts", func(t *testing.T) {
			in := RedocOpts{
				BasePath: "a",
				Path:     "b",
				SpecURL:  "c",
				Template: "d",
				Title:    "e",
				RedocURL: "f",
			}
			out := ToCommonUIOptions(in)

			require.EqualT(t, "a", out.BasePath)
			require.EqualT(t, "b", out.Path)
			require.EqualT(t, "c", out.SpecURL)
			require.EqualT(t, "d", out.Template)
			require.EqualT(t, "e", out.Title)
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
			out := ToCommonUIOptions(in)

			require.EqualT(t, "a", out.BasePath)
			require.EqualT(t, "b", out.Path)
			require.EqualT(t, "c", out.SpecURL)
			require.EqualT(t, "d", out.Template)
			require.EqualT(t, "e", out.Title)
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
			out := ToCommonUIOptions(in)

			require.EqualT(t, "a", out.BasePath)
			require.EqualT(t, "b", out.Path)
			require.EqualT(t, "c", out.SpecURL)
			require.EqualT(t, "d", out.Template)
			require.EqualT(t, "e", out.Title)
		})
	})

	t.Run("from UIOptions to any UI options", func(t *testing.T) {
		in := UIOptions{
			BasePath: "a",
			Path:     "b",
			SpecURL:  "c",
			Template: "d",
			Title:    "e",
		}

		t.Run("to RedocOpts", func(t *testing.T) {
			var out RedocOpts
			FromCommonToAnyOptions(in, &out)
			require.EqualT(t, "a", out.BasePath)
			require.EqualT(t, "b", out.Path)
			require.EqualT(t, "c", out.SpecURL)
			require.EqualT(t, "d", out.Template)
			require.EqualT(t, "e", out.Title)
		})

		t.Run("to RapiDocOpts", func(t *testing.T) {
			var out RapiDocOpts
			FromCommonToAnyOptions(in, &out)
			require.EqualT(t, "a", out.BasePath)
			require.EqualT(t, "b", out.Path)
			require.EqualT(t, "c", out.SpecURL)
			require.EqualT(t, "d", out.Template)
			require.EqualT(t, "e", out.Title)
		})

		t.Run("to SwaggerUIOpts", func(t *testing.T) {
			var out SwaggerUIOpts
			FromCommonToAnyOptions(in, &out)
			require.EqualT(t, "a", out.BasePath)
			require.EqualT(t, "b", out.Path)
			require.EqualT(t, "c", out.SpecURL)
			require.EqualT(t, "d", out.Template)
			require.EqualT(t, "e", out.Title)
		})
	})
}
