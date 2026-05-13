// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package mediatype

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// Alias fixtures. The canonical YAML target is mtYAML
// ("application/yaml"), defined in mediatype.go and visible here
// because both files share the package.
const (
	mtXYAML     = "application/x-yaml"
	mtTextYAML  = "text/yaml"
	mtTextXYAML = "text/x-yaml"

	// Parametric forms used in the param-aware Match / BestMatch
	// tests. The space after ';' matches what mime.FormatMediaType
	// emits, so these can be used both as Parse inputs and as
	// String() output assertions.
	mtXYAMLUTF8 = "application/x-yaml; charset=utf-8"
	mtYAMLUTF8  = "application/yaml; charset=utf-8"
	mtYAMLAscii = "application/yaml; charset=ascii"
)

func TestAliasesTable(t *testing.T) {
	// Pin the table contents — additions need a corresponding RFC
	// citation in the commit message and a matching update here.
	require.Len(t, aliases, 3)
	for _, alias := range []string{mtXYAML, mtTextYAML, mtTextXYAML} {
		canon, ok := aliases[alias]
		require.TrueT(t, ok, "missing alias %q", alias)
		assert.EqualT(t, mtYAML, canon, "alias %q canonical", alias)
	}

	// Acyclic: no value may also appear as a key. Otherwise
	// Canonical() would need fixpoint iteration.
	for _, canon := range aliases {
		_, isAlsoKey := aliases[canon]
		assert.FalseT(t, isAlsoKey, "canonical %q must not also be an alias key", canon)
	}
}

func TestCanonical(t *testing.T) {
	t.Run("known alias resolves to canonical", func(t *testing.T) {
		cases := []string{
			mtXYAML,
			mtTextYAML,
			mtTextXYAML,
		}
		for _, s := range cases {
			t.Run(s, func(t *testing.T) {
				m, err := Parse(s)
				require.NoError(t, err)
				got := m.Canonical()
				assert.EqualT(t, tApp, got.Type)
				assert.EqualT(t, subtypeYAML, got.Subtype)
				assert.EqualT(t, "", got.Suffix)
			})
		}
	})

	t.Run("non-alias returned unchanged", func(t *testing.T) {
		cases := []string{
			jsonMime,
			mtYAML, // already canonical
			textPlain,
			starStar,
			"application/vnd.api+json",
		}
		for _, s := range cases {
			t.Run(s, func(t *testing.T) {
				m, err := Parse(s)
				require.NoError(t, err)
				got := m.Canonical()
				assert.EqualT(t, m.Type, got.Type)
				assert.EqualT(t, m.Subtype, got.Subtype)
				assert.EqualT(t, m.Suffix, got.Suffix)
			})
		}
	})

	t.Run("params and q preserved", func(t *testing.T) {
		m, err := Parse("application/x-yaml; version=1; q=0.7")
		require.NoError(t, err)
		got := m.Canonical()
		assert.EqualT(t, tApp, got.Type)
		assert.EqualT(t, subtypeYAML, got.Subtype)
		assert.EqualT(t, "", got.Suffix)
		assert.EqualValues(t, map[string]string{"version": "1"}, got.Params)
		assert.EqualT(t, 0.7, got.Q)
	})

	t.Run("does not mutate receiver", func(t *testing.T) {
		m, err := Parse("application/x-yaml; version=1")
		require.NoError(t, err)
		origType := m.Type
		origSub := m.Subtype

		_ = m.Canonical()

		assert.EqualT(t, origType, m.Type)
		assert.EqualT(t, origSub, m.Subtype)
	})
}

func TestMediaType_Match(t *testing.T) {
	cases := []struct {
		name     string
		bound    string
		other    string
		wantKind MatchKind
	}{
		// Exact tier — direct or via wildcards.
		{"identical", jsonMime, jsonMime, MatchExact},
		{"bound wildcard", starStar, jsonMime, MatchExact},
		{"other wildcard", jsonMime, starStar, MatchExact},
		{"both already canonical", mtYAML, mtYAML, MatchExact},
		{"both already same alias", mtXYAML, mtXYAML, MatchExact},

		// Alias tier — only the alias bridge makes them agree.
		{"canonical vs x-prefixed", mtYAML, mtXYAML, MatchAlias},
		{"x-prefixed vs canonical", mtXYAML, mtYAML, MatchAlias},
		{"text yaml vs canonical", mtTextYAML, mtYAML, MatchAlias},
		{"text x-yaml vs canonical", mtTextXYAML, mtYAML, MatchAlias},
		{"two different aliases same canonical", mtXYAML, mtTextYAML, MatchAlias},

		// Alias subtype with params that fully agree on both sides
		// — the bridge still applies. Pins the case where a request
		// for "application/yaml; charset=utf-8" sees an offered
		// "application/x-yaml; charset=utf-8" as a match.
		{"alias subtype with matching params", mtXYAMLUTF8, mtYAMLUTF8, MatchAlias},

		// None tier.
		{"unrelated types", jsonMime, textPlain, MatchNone},
		{"unrelated and unaliased", jsonMime, mtXYAML, MatchNone},
		{"alias plus param mismatch",
			"application/x-yaml; version=1",
			"application/yaml; version=2", MatchNone},
		// Exact canonical subtype but disagreeing params: the
		// param-subset rule trumps the subtype agreement, so this
		// is NOT a match — neither exact nor alias. Documents that
		// charset (and other params) bind even when the subtype
		// would otherwise agree.
		{"exact subtype with param mismatch",
			mtYAMLAscii, mtYAMLUTF8, MatchNone},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b, err := Parse(c.bound)
			require.NoError(t, err)
			o, err := Parse(c.other)
			require.NoError(t, err)
			assert.EqualT(t, c.wantKind, b.Match(o),
				"%q.Match(%q)", c.bound, c.other)
		})
	}
}

func TestBestMatch_AliasTier(t *testing.T) {
	t.Run("exact beats alias regardless of offer order", func(t *testing.T) {
		// Accept: application/yaml; offers list the alias first, then
		// the canonical. The canonical should still win on tier.
		accept := ParseAccept(mtYAML)
		offers := Set{
			mustParse(t, mtXYAML),
			mustParse(t, mtYAML),
		}
		best, ok := accept.BestMatch(offers)
		require.TrueT(t, ok)
		assert.EqualT(t, mtYAML, best.String())
	})

	t.Run("exact beats alias when alias is first", func(t *testing.T) {
		// Same idea with all three aliases in front.
		accept := ParseAccept(mtYAML)
		offers := Set{
			mustParse(t, mtXYAML),
			mustParse(t, mtTextYAML),
			mustParse(t, mtTextXYAML),
			mustParse(t, mtYAML),
		}
		best, ok := accept.BestMatch(offers)
		require.TrueT(t, ok)
		assert.EqualT(t, mtYAML, best.String())
	})

	t.Run("alias-only offers still match", func(t *testing.T) {
		accept := ParseAccept(mtYAML)
		offers := Set{
			mustParse(t, mtXYAML),
		}
		best, ok := accept.BestMatch(offers)
		require.TrueT(t, ok)
		assert.EqualT(t, mtXYAML, best.String())
	})

	t.Run("first alias wins among alias-only offers", func(t *testing.T) {
		accept := ParseAccept(mtYAML)
		offers := Set{
			mustParse(t, mtTextYAML),
			mustParse(t, mtXYAML),
		}
		best, ok := accept.BestMatch(offers)
		require.TrueT(t, ok)
		assert.EqualT(t, mtTextYAML, best.String())
	})

	t.Run("q dominates alias tier", func(t *testing.T) {
		// Accept: text/plain;q=0.1, application/yaml. Offered:
		// text/plain (exact match to a low-q entry),
		// application/x-yaml (alias match to a high-q entry).
		// The alias match wins because q is higher.
		accept := ParseAccept("text/plain;q=0.1, application/yaml")
		offers := Set{
			mustParse(t, textPlain),
			mustParse(t, mtXYAML),
		}
		best, ok := accept.BestMatch(offers)
		require.TrueT(t, ok)
		assert.EqualT(t, mtXYAML, best.String())
	})

	t.Run("specificity dominates alias tier", func(t *testing.T) {
		// Accept lists both */* and application/yaml at q=1. The
		// alias offer matches application/yaml via the alias bridge
		// (SpecificityExact). A second offer matches only */*
		// (SpecificityAny). Specificity wins, so the alias offer is
		// picked.
		accept := ParseAccept("*/*, application/yaml")
		offers := Set{
			mustParse(t, textPlain), // matches only via */*
			mustParse(t, mtXYAML),   // matches application/yaml via alias
		}
		best, ok := accept.BestMatch(offers)
		require.TrueT(t, ok)
		assert.EqualT(t, mtXYAML, best.String())
	})

	t.Run("unrelated offers don't match", func(t *testing.T) {
		accept := ParseAccept(mtYAML)
		offers := Set{
			mustParse(t, jsonMime),
			mustParse(t, textPlain),
		}
		_, ok := accept.BestMatch(offers)
		assert.FalseT(t, ok)
	})

	t.Run("alias with matching params wins when exact subtype disagrees on params", func(t *testing.T) {
		// Pins the scenario:
		//   accept: application/yaml; charset=utf-8
		//   offers:
		//     - application/x-yaml; charset=utf-8  (alias subtype, params agree)
		//     - application/yaml;   charset=ascii  (exact subtype, params disagree)
		//
		// Offer 2 looks like the "more direct" candidate, but the
		// param-subset rule fails (ascii ≠ utf-8) and it doesn't
		// match at all. Offer 1 is the only remaining candidate and
		// wins via the alias bridge with full param agreement.
		accept := ParseAccept(mtYAMLUTF8)
		offers := Set{
			mustParse(t, mtXYAMLUTF8),
			mustParse(t, mtYAMLAscii),
		}
		best, ok := accept.BestMatch(offers)
		require.TrueT(t, ok)
		assert.EqualT(t, mtXYAMLUTF8, best.String())
	})
}

func TestMatchFirst_AliasTier(t *testing.T) {
	t.Run("exact in pass 1 even when later than alias", func(t *testing.T) {
		// allowed lists the alias first, then the canonical. Pass 1
		// scans the whole list looking for exact — the canonical
		// matches and wins.
		allowed := []string{mtXYAML, mtYAML}
		got, ok, err := MatchFirst(allowed, mtYAML)
		require.NoError(t, err)
		require.TrueT(t, ok)
		assert.EqualT(t, mtYAML, got.String())
	})

	t.Run("alias fallback in pass 2", func(t *testing.T) {
		// No exact match anywhere; the alias bridge picks up the
		// first alias in the list.
		allowed := []string{textPlain, mtXYAML, mtTextYAML}
		got, ok, err := MatchFirst(allowed, mtYAML)
		require.NoError(t, err)
		require.TrueT(t, ok)
		assert.EqualT(t, mtXYAML, got.String())
	})

	t.Run("first alias wins within tier", func(t *testing.T) {
		// Two aliases, neither exact. Pass 2 returns the earliest.
		allowed := []string{mtTextYAML, mtXYAML}
		got, ok, err := MatchFirst(allowed, mtYAML)
		require.NoError(t, err)
		require.TrueT(t, ok)
		assert.EqualT(t, mtTextYAML, got.String())
	})

	t.Run("no match returns false", func(t *testing.T) {
		allowed := []string{textPlain, jsonMime}
		_, ok, err := MatchFirst(allowed, mtYAML)
		require.NoError(t, err)
		assert.FalseT(t, ok)
	})
}
