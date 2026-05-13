// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package mediatype

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// Test fixtures local to the suffix tests. The package-wide fixtures
// (tApp, tJSON, …) live in mediatype_test.go.
const (
	tXML  = "xml"
	tYAML = "yaml"

	subAPIJSON  = "vnd.api+json"
	subProbJSON = "problem+json"
	subFooXML   = "vnd.foo+xml"
	subFooYAML  = "vnd.foo+yaml"
	subFooBar   = "foo+bar+json"
	subPlusJSON = "+json"

	mtAPIJSON      = "application/vnd.api+json"
	mtProbJSON     = "application/problem+json"
	mtProbJSONUTF8 = "application/problem+json;charset=utf-8"
	mtFooBar       = "application/foo+bar+json"
	mtPlusJSON     = "application/+json"
)

func TestParseSuffix(t *testing.T) {
	cases := []struct {
		in      string
		wantSub string
		wantSfx string
	}{
		// No suffix.
		{jsonMime, tJSON, ""},
		{textPlain, tPlain, ""},
		{"application/json;charset=utf-8", tJSON, ""},
		{starStar, "*", ""},
		{"application/*", "*", ""},

		// Single suffix — RFC 6839 happy path.
		{mtAPIJSON, subAPIJSON, tJSON},
		{mtProbJSON, subProbJSON, tJSON},
		{"application/ld+json", "ld+json", tJSON},
		{"application/geo+json", "geo+json", tJSON},
		{"application/hal+json", "hal+json", tJSON},
		{"application/vnd.foo+xml", subFooXML, tXML},
		{"application/vnd.foo+yaml", subFooYAML, tYAML},

		// Suffix with parameters.
		{mtProbJSONUTF8, subProbJSON, tJSON},
		{"application/vnd.api+json; version=1", subAPIJSON, tJSON},

		// Case-insensitivity: mime.ParseMediaType lowercases the
		// subtype, so the suffix is recovered in lowercase too.
		{"Application/Vnd.Api+JSON", subAPIJSON, tJSON},
		{"APPLICATION/VND.FOO+XML", subFooXML, tXML},

		// Multiple '+': only the trailing token is the suffix.
		{mtFooBar, subFooBar, tJSON},
		{"application/a+b+c+xml", "a+b+c+xml", tXML},

		// Bare suffix (no name before '+'): defensive — not strictly
		// valid per RFC 6839, but the trailing token is still a
		// well-defined suffix.
		{mtPlusJSON, subPlusJSON, tJSON},

		// Trailing '+' with no token → no suffix.
		{"application/json+", "json+", ""},
		{"application/+", "+", ""},

		// Unknown suffix is still parsed as a suffix (Base() decides
		// whether it has a known base).
		{"application/vnd.foo+cbor", "vnd.foo+cbor", "cbor"},
		{"application/vnd.foo+zip", "vnd.foo+zip", "zip"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, err := Parse(c.in)
			require.NoError(t, err)
			assert.EqualT(t, c.wantSub, got.Subtype, "Subtype")
			assert.EqualT(t, c.wantSfx, got.Suffix, "Suffix")
		})
	}
}

func TestBase(t *testing.T) {
	t.Run("unsuffixed types return receiver unchanged", func(t *testing.T) {
		cases := []string{
			jsonMime,
			textPlainUTF8,
			starStar,
			"application/*",
		}
		for _, s := range cases {
			t.Run(s, func(t *testing.T) {
				m, err := Parse(s)
				require.NoError(t, err)
				got := m.Base()
				assert.EqualT(t, m.Type, got.Type)
				assert.EqualT(t, m.Subtype, got.Subtype)
				assert.EqualT(t, m.Suffix, got.Suffix)
			})
		}
	})

	t.Run("known suffix resolves to base type", func(t *testing.T) {
		cases := []struct {
			in         string
			wantType   string
			wantSubtyp string
		}{
			{mtAPIJSON, tApp, tJSON},
			{mtProbJSON, tApp, tJSON},
			{"application/ld+json", tApp, tJSON},
			{"application/geo+json", tApp, tJSON},
			{"application/hal+json", tApp, tJSON},
			{"application/vnd.foo+xml", tApp, tXML},
			{"application/vnd.foo+yaml", tApp, tYAML},
			{mtFooBar, tApp, tJSON},
			{mtPlusJSON, tApp, tJSON},
			{"Application/Vnd.Api+JSON", tApp, tJSON},
		}
		for _, c := range cases {
			t.Run(c.in, func(t *testing.T) {
				m, err := Parse(c.in)
				require.NoError(t, err)
				got := m.Base()
				assert.EqualT(t, c.wantType, got.Type)
				assert.EqualT(t, c.wantSubtyp, got.Subtype)
				assert.EqualT(t, "", got.Suffix, "Base() result has no suffix")
			})
		}
	})

	t.Run("unknown suffix returns receiver unchanged", func(t *testing.T) {
		cases := []string{
			"application/vnd.foo+cbor",
			"application/vnd.foo+zip",
			"application/vnd.foo+ber",
		}
		for _, s := range cases {
			t.Run(s, func(t *testing.T) {
				m, err := Parse(s)
				require.NoError(t, err)
				got := m.Base()
				assert.EqualT(t, m.Type, got.Type)
				assert.EqualT(t, m.Subtype, got.Subtype)
				assert.EqualT(t, m.Suffix, got.Suffix)
			})
		}
	})

	t.Run("base drops params and q", func(t *testing.T) {
		m, err := Parse("application/problem+json; charset=utf-8; q=0.7")
		require.NoError(t, err)
		require.NotNil(t, m.Params)
		require.EqualT(t, 0.7, m.Q)

		got := m.Base()
		assert.Nil(t, got.Params, "Base() drops params")
		assert.EqualT(t, 0.0, got.Q, "Base() drops q-value")
		assert.EqualT(t, tApp, got.Type)
		assert.EqualT(t, tJSON, got.Subtype)
	})

	t.Run("base does not mutate receiver", func(t *testing.T) {
		m, err := Parse("application/problem+json; charset=utf-8")
		require.NoError(t, err)
		origSub := m.Subtype
		origSfx := m.Suffix
		origParams := m.Params

		_ = m.Base()

		assert.EqualT(t, origSub, m.Subtype)
		assert.EqualT(t, origSfx, m.Suffix)
		assert.EqualValues(t, origParams, m.Params)
	})
}

func TestSuffixRoundtrip(t *testing.T) {
	// String() must continue to round-trip media types carrying a
	// suffix — the suffix is part of Subtype on the wire and is not
	// emitted separately.
	inputs := []string{
		mtAPIJSON,
		mtProbJSONUTF8,
		mtFooBar,
		mtPlusJSON,
	}
	for _, s := range inputs {
		t.Run(s, func(t *testing.T) {
			m1, err := Parse(s)
			require.NoError(t, err)
			m2, err := Parse(m1.String())
			require.NoError(t, err)
			assert.EqualT(t, m1.Type, m2.Type)
			assert.EqualT(t, m1.Subtype, m2.Subtype)
			assert.EqualT(t, m1.Suffix, m2.Suffix)
			assert.EqualValues(t, m1.Params, m2.Params)
		})
	}
}

func TestSuffixBaseTable(t *testing.T) {
	// The table is small and explicit; assert its contents to catch
	// accidental edits. MediaType is not comparable (has a map field),
	// so the rows are checked component-wise.
	require.Len(t, SuffixBase, 3)
	for suffix, wantSub := range map[string]string{
		tJSON: tJSON,
		tXML:  tXML,
		tYAML: tYAML,
	} {
		got, ok := SuffixBase[suffix]
		require.TrueT(t, ok, "missing entry for %q", suffix)
		assert.EqualT(t, tApp, got.Type, "%q.Type", suffix)
		assert.EqualT(t, wantSub, got.Subtype, "%q.Subtype", suffix)
		assert.EqualT(t, "", got.Suffix, "%q.Suffix", suffix)
		assert.Nil(t, got.Params, "%q.Params", suffix)
	}
}

func TestStripParamsPreservesSuffix(t *testing.T) {
	// StripParams is a sibling derived-value helper; confirm it
	// carries Suffix through.
	m, err := Parse("application/vnd.api+json;charset=utf-8;q=0.5")
	require.NoError(t, err)
	require.EqualT(t, tJSON, m.Suffix)

	stripped := m.StripParams()
	assert.Nil(t, stripped.Params)
	assert.EqualT(t, subAPIJSON, stripped.Subtype)
	assert.EqualT(t, tJSON, stripped.Suffix)
	assert.EqualT(t, 0.5, stripped.Q)
}
