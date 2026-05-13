// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package mediatype

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
)

// Sentinel codec names used as map values in the Lookup tests.
const (
	codecJSON = "json-codec"
	codecYAML = "yaml-codec"
)

func TestLookup(t *testing.T) {
	t.Run("empty map", func(t *testing.T) {
		got, ok := Lookup(map[string]int(nil), jsonMime)
		assert.FalseT(t, ok)
		assert.EqualT(t, 0, got)
	})

	t.Run("tier 1 — raw key hit", func(t *testing.T) {
		m := map[string]string{jsonMime: codecJSON}
		got, ok := Lookup(m, jsonMime)
		assert.TrueT(t, ok)
		assert.EqualT(t, codecJSON, got)
	})

	t.Run("tier 2 — parameters stripped", func(t *testing.T) {
		// Content-Type with charset: map key has no params.
		m := map[string]string{jsonMime: codecJSON}
		got, ok := Lookup(m, "application/json; charset=utf-8")
		assert.TrueT(t, ok)
		assert.EqualT(t, codecJSON, got)
	})

	t.Run("tier 2 — case normalized via Parse", func(t *testing.T) {
		m := map[string]string{jsonMime: codecJSON}
		got, ok := Lookup(m, "Application/JSON")
		assert.TrueT(t, ok)
		assert.EqualT(t, codecJSON, got)
	})

	t.Run("tier 3 — alias bridge (request canonical, map aliased)", func(t *testing.T) {
		// Map registered under x-yaml; request asks for yaml.
		m := map[string]string{mtXYAML: codecYAML}
		got, ok := Lookup(m, mtYAML)
		assert.TrueT(t, ok)
		assert.EqualT(t, codecYAML, got)
	})

	t.Run("tier 3 — alias bridge (request aliased, map canonical)", func(t *testing.T) {
		// Map registered under canonical yaml; request uses an alias.
		m := map[string]string{mtYAML: codecYAML}
		got, ok := Lookup(m, mtXYAML)
		assert.TrueT(t, ok)
		assert.EqualT(t, codecYAML, got)
	})

	t.Run("tier 3 — two different aliases of same canonical", func(t *testing.T) {
		// Map registered under text/yaml; request uses x-yaml. Both
		// canonicalize to application/yaml.
		m := map[string]string{mtTextYAML: codecYAML}
		got, ok := Lookup(m, mtXYAML)
		assert.TrueT(t, ok)
		assert.EqualT(t, codecYAML, got)
	})

	t.Run("tier 3 — alias with params on request", func(t *testing.T) {
		m := map[string]string{mtYAML: codecYAML}
		got, ok := Lookup(m, "application/x-yaml; version=1")
		assert.TrueT(t, ok)
		assert.EqualT(t, codecYAML, got)
	})

	t.Run("exact wins over alias", func(t *testing.T) {
		// Both forms registered; raw key takes precedence (tier 1).
		m := map[string]string{
			mtYAML:  "canonical-codec",
			mtXYAML: "alias-codec",
		}
		got, ok := Lookup(m, mtYAML)
		assert.TrueT(t, ok)
		assert.EqualT(t, "canonical-codec", got)

		got, ok = Lookup(m, mtXYAML)
		assert.TrueT(t, ok)
		assert.EqualT(t, "alias-codec", got)
	})

	t.Run("does not fall back to wildcard", func(t *testing.T) {
		// Lookup deliberately does not consult "*/*"; the caller
		// must do that explicitly if desired.
		m := map[string]string{"*/*": "catch-all"}
		_, ok := Lookup(m, jsonMime)
		assert.FalseT(t, ok)
	})

	t.Run("malformed media type returns false", func(t *testing.T) {
		m := map[string]string{jsonMime: codecJSON}
		_, ok := Lookup(m, "not a media type")
		assert.FalseT(t, ok)
	})

	t.Run("unrelated type misses cleanly", func(t *testing.T) {
		m := map[string]string{jsonMime: codecJSON}
		_, ok := Lookup(m, textPlain)
		assert.FalseT(t, ok)
	})

	t.Run("works with non-pointer value types", func(t *testing.T) {
		// Smoke test for the generic: any value type works.
		type codec struct{ name string }
		m := map[string]codec{jsonMime: {name: codecJSON}}
		got, ok := Lookup(m, "application/json; charset=utf-8")
		assert.TrueT(t, ok)
		assert.EqualT(t, codecJSON, got.name)
	})
}
