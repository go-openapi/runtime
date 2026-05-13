// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package mediatype

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestMatchKindOrdering(t *testing.T) {
	// MatchKind values must be ordered None < Suffix < Alias < Exact
	// so the "stronger tier wins" comparisons in BestMatch and
	// MatchFirst stay correct.
	assert.TrueT(t, MatchNone < MatchSuffix, "None < Suffix")
	assert.TrueT(t, MatchSuffix < MatchAlias, "Suffix < Alias")
	assert.TrueT(t, MatchAlias < MatchExact, "Alias < Exact")
}

func TestMediaType_Match_suffix(t *testing.T) {
	t.Run("vendor +json vs canonical json", func(t *testing.T) {
		b := mustParse(t, mtAPIJSON)
		o := mustParse(t, jsonMime)
		assert.EqualT(t, MatchSuffix, b.Match(o))
		assert.EqualT(t, MatchSuffix, o.Match(b))
	})

	t.Run("problem+json vs json", func(t *testing.T) {
		b := mustParse(t, mtProbJSON)
		o := mustParse(t, jsonMime)
		assert.EqualT(t, MatchSuffix, b.Match(o))
	})

	t.Run("vendor +xml vs canonical xml", func(t *testing.T) {
		b := mustParse(t, "application/vnd.foo+xml")
		o := mustParse(t, "application/xml")
		assert.EqualT(t, MatchSuffix, b.Match(o))
	})

	t.Run("vendor +yaml folds via suffix+alias to x-yaml", func(t *testing.T) {
		// +yaml suffix folds to application/yaml (canonical, per RFC 9512);
		// application/x-yaml alias folds to application/yaml. Both sides
		// resolve to the same canonical → MatchSuffix.
		b := mustParse(t, mtFooYAML)
		o := mustParse(t, mtXYAML)
		assert.EqualT(t, MatchSuffix, b.Match(o))
	})

	t.Run("unrelated suffix bases don't bridge", func(t *testing.T) {
		// +json and +xml fold to different bases → no match.
		b := mustParse(t, mtAPIJSON)
		o := mustParse(t, "application/vnd.foo+xml")
		assert.EqualT(t, MatchNone, b.Match(o))
	})
}

func TestBestMatch_SuffixTier(t *testing.T) {
	t.Run("strict default ignores suffix matches", func(t *testing.T) {
		accept := ParseAccept(jsonMime)
		offers := Set{mustParse(t, mtAPIJSON)}
		_, ok := accept.BestMatch(offers)
		assert.FalseT(t, ok, "default BestMatch must not pick a suffix-only offer")
	})

	t.Run("AllowSuffix counts suffix matches", func(t *testing.T) {
		accept := ParseAccept(jsonMime)
		offers := Set{mustParse(t, mtAPIJSON)}
		best, ok := accept.BestMatch(offers, AllowSuffix())
		require.TrueT(t, ok)
		assert.EqualT(t, mtAPIJSON, best.String())
	})

	t.Run("AllowSuffix: exact beats alias beats suffix", func(t *testing.T) {
		// Accept: application/yaml. Offers: vendor+yaml (suffix tier),
		// application/x-yaml (alias tier), application/yaml (exact).
		// Exact must win regardless of offer order.
		accept := ParseAccept(mtYAML)
		offers := Set{
			mustParse(t, mtFooYAML),
			mustParse(t, mtXYAML),
			mustParse(t, mtYAML),
		}
		best, ok := accept.BestMatch(offers, AllowSuffix())
		require.TrueT(t, ok)
		assert.EqualT(t, mtYAML, best.String())
	})

	t.Run("AllowSuffix: alias beats suffix", func(t *testing.T) {
		// Accept: application/yaml. Offers: vendor+yaml (suffix),
		// application/x-yaml (alias). Alias wins.
		accept := ParseAccept(mtYAML)
		offers := Set{
			mustParse(t, mtFooYAML),
			mustParse(t, mtXYAML),
		}
		best, ok := accept.BestMatch(offers, AllowSuffix())
		require.TrueT(t, ok)
		assert.EqualT(t, mtXYAML, best.String())
	})

	t.Run("AllowSuffix: first suffix wins among suffix-only offers", func(t *testing.T) {
		accept := ParseAccept(jsonMime)
		offers := Set{
			mustParse(t, mtProbJSON),
			mustParse(t, mtAPIJSON),
		}
		best, ok := accept.BestMatch(offers, AllowSuffix())
		require.TrueT(t, ok)
		assert.EqualT(t, mtProbJSON, best.String())
	})

	t.Run("AllowSuffix: q dominates the suffix tier", func(t *testing.T) {
		// Accept: text/plain;q=0.1, application/json. Offered:
		// text/plain (exact match to low-q entry),
		// application/vnd.api+json (suffix match to high-q entry).
		// Suffix tier wins because q is higher.
		accept := ParseAccept("text/plain;q=0.1, application/json")
		offers := Set{
			mustParse(t, textPlain),
			mustParse(t, mtAPIJSON),
		}
		best, ok := accept.BestMatch(offers, AllowSuffix())
		require.TrueT(t, ok)
		assert.EqualT(t, mtAPIJSON, best.String())
	})
}

func TestMatchFirst_SuffixTier(t *testing.T) {
	t.Run("strict default ignores suffix matches", func(t *testing.T) {
		_, ok, err := MatchFirst([]string{mtAPIJSON}, jsonMime)
		require.NoError(t, err)
		assert.FalseT(t, ok)
	})

	t.Run("AllowSuffix counts suffix matches", func(t *testing.T) {
		got, ok, err := MatchFirst([]string{mtAPIJSON}, jsonMime, AllowSuffix())
		require.NoError(t, err)
		require.TrueT(t, ok)
		assert.EqualT(t, mtAPIJSON, got.String())
	})

	t.Run("AllowSuffix: exact wins even when listed after a suffix", func(t *testing.T) {
		// Three-pass scan: pass 1 (exact) finds jsonMime even though
		// it comes after a suffix candidate.
		got, ok, err := MatchFirst([]string{mtAPIJSON, jsonMime}, jsonMime, AllowSuffix())
		require.NoError(t, err)
		require.TrueT(t, ok)
		assert.EqualT(t, jsonMime, got.String())
	})

	t.Run("AllowSuffix: alias wins over suffix", func(t *testing.T) {
		got, ok, err := MatchFirst([]string{mtFooYAML, mtXYAML}, mtYAML, AllowSuffix())
		require.NoError(t, err)
		require.TrueT(t, ok)
		assert.EqualT(t, mtXYAML, got.String())
	})
}

func TestLookup_SuffixTier(t *testing.T) {
	t.Run("strict default — query has suffix, map has base → miss", func(t *testing.T) {
		m := map[string]string{jsonMime: codecJSON}
		_, ok := Lookup(m, mtAPIJSON)
		assert.FalseT(t, ok, "default Lookup must not fall back to suffix base")
	})

	t.Run("AllowSuffix — query has suffix, map has base → hit", func(t *testing.T) {
		m := map[string]string{jsonMime: codecJSON}
		got, ok := Lookup(m, mtAPIJSON, AllowSuffix())
		require.TrueT(t, ok)
		assert.EqualT(t, codecJSON, got)
	})

	t.Run("AllowSuffix — problem+json folds to JSON consumer", func(t *testing.T) {
		m := map[string]string{jsonMime: codecJSON}
		got, ok := Lookup(m, mtProbJSON, AllowSuffix())
		require.TrueT(t, ok)
		assert.EqualT(t, codecJSON, got)
	})

	t.Run("AllowSuffix — suffix folds, then alias bridge picks up x-yaml registration", func(t *testing.T) {
		// Map keyed by application/x-yaml (the legacy alias). Query
		// is application/vnd.foo+yaml. Tier 5 folds the suffix to
		// application/yaml, then the alias-canonical of that key
		// (application/x-yaml from the aliases map's value lookup —
		// but aliases maps alias→canonical, so the inverse is not in
		// the table; we need the map-side iteration done in tier 4
		// to catch the x-yaml registration). Since tier 4 runs
		// before tier 5, this case is handled by tier 4 once the
		// query canonical is application/yaml — but the query
		// canonical is the vendor mime, not the base. So tier 5 has
		// to do its own alias canonicalization of the base.
		m := map[string]string{mtXYAML: codecYAML}
		got, ok := Lookup(m, mtFooYAML, AllowSuffix())
		require.TrueT(t, ok)
		assert.EqualT(t, codecYAML, got)
	})

	t.Run("AllowSuffix — no backwards fold (only vendor registered, base queried)", func(t *testing.T) {
		// The inverse of the main case: the only registered consumer
		// is the vendor type; the query is plain JSON. Lookup must
		// NOT pick the vendor consumer — that would be a wider
		// tolerance than asked for.
		m := map[string]string{mtAPIJSON: codecJSON}
		_, ok := Lookup(m, jsonMime, AllowSuffix())
		assert.FalseT(t, ok)
	})

	t.Run("AllowSuffix — unknown suffix doesn't fall back", func(t *testing.T) {
		// +cbor is not in the suffixBase table, so Base() returns
		// the receiver unchanged → no suffix tier fire.
		m := map[string]string{jsonMime: codecJSON}
		_, ok := Lookup(m, "application/vnd.foo+cbor", AllowSuffix())
		assert.FalseT(t, ok)
	})

	t.Run("AllowSuffix — exact / alias tiers still win first", func(t *testing.T) {
		// Map has both vnd.api+json (exact match candidate) and json
		// (suffix base). Query is vnd.api+json. Tier 1 hits before
		// tier 5 fires.
		m := map[string]string{
			jsonMime:  codecJSON,
			mtAPIJSON: "vendor-codec",
		}
		got, ok := Lookup(m, mtAPIJSON, AllowSuffix())
		require.TrueT(t, ok)
		assert.EqualT(t, "vendor-codec", got)
	})
}
