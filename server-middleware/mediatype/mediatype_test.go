// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package mediatype

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// Test fixtures: extracted to dedup goconst hits in the table-driven cases.
const (
	textPlain     = "text/plain"
	textPlainUTF8 = "text/plain;charset=utf-8"
	textHTML      = "text/html"
	textWild      = "text/*"
	starStar      = "*/*"
	imagePNG      = "image/png"
	imageJPG      = "image/jpg"
	imageGIF      = "image/gif"
	imageWild     = "image/*"
	jsonMime      = "application/json"
	xy            = "x/y"
	htmlPNGq05    = "text/html, image/png; q=0.5"
	html05PNG     = "text/html;q=0.5, image/png"
	pngWildq05    = "image/png, image/*;q=0.5"
	pngWild       = "image/png, image/*"

	// Component fragments referenced as expected values.
	tApp     = "application"
	tText    = "text"
	tHTML    = "html"
	tJSON    = "json"
	tPlain   = "plain"
	pCharset = "charset"
	vUTF8    = "utf-8"
)

func TestParse(t *testing.T) {
	t.Run("happy paths", func(t *testing.T) {
		cases := []struct {
			in       string
			wantType string
			wantSub  string
			wantSfx  string
			wantQ    float64
			wantPars map[string]string
		}{
			{jsonMime, tApp, tJSON, "", 1.0, nil},
			{"  Application/JSON  ", tApp, tJSON, "", 1.0, nil},
			{textPlainUTF8, tText, tPlain, "", 1.0, map[string]string{pCharset: vUTF8}},
			{"text/plain; charset=utf-8", tText, tPlain, "", 1.0, map[string]string{pCharset: vUTF8}},
			{"text/html;q=0.5", tText, tHTML, "", 0.5, nil},
			{"text/html;q=0", tText, tHTML, "", 0.0, nil},
			{"text/html;q=1", tText, tHTML, "", 1.0, nil},
			{"text/html;q=0.7;version=2", tText, tHTML, "", 0.7, map[string]string{"version": "2"}},
			{textWild, tText, "*", "", 1.0, nil},
			{starStar, "*", "*", "", 1.0, nil},
			// RFC 6839 structured-syntax suffix coverage. Exhaustive
			// suffix cases live in mediatype_suffix_test.go; here we
			// just confirm Parse populates Suffix alongside the other
			// fields on a representative sample.
			{mtAPIJSON, tApp, subAPIJSON, tJSON, 1.0, nil},
			{mtProbJSONUTF8, tApp, subProbJSON, tJSON, 1.0, map[string]string{pCharset: vUTF8}},
			{"application/vnd.foo+xml;q=0.5", tApp, subFooXML, tXML, 0.5, nil},
		}
		for _, c := range cases {
			t.Run(c.in, func(t *testing.T) {
				got, err := Parse(c.in)
				require.NoError(t, err)
				assert.EqualT(t, c.wantType, got.Type)
				assert.EqualT(t, c.wantSub, got.Subtype)
				assert.EqualT(t, c.wantSfx, got.Suffix)
				assert.EqualT(t, c.wantQ, got.Q)
				if c.wantPars == nil {
					assert.Nil(t, got.Params)
				} else {
					assert.EqualValues(t, c.wantPars, got.Params)
				}
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		invalid := []string{
			"",
			"   ",
			tApp,
			"application/",
			"/json",
			"application(",
			"application/json;char*",
		}
		for _, s := range invalid {
			t.Run(s, func(t *testing.T) {
				_, err := Parse(s)
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrMalformed)
			})
		}
	})

	t.Run("q clamped to [0,1]", func(t *testing.T) {
		got, err := Parse("text/plain;q=2")
		require.NoError(t, err)
		assert.EqualT(t, 1.0, got.Q)

		got, err = Parse("text/plain;q=-0.5")
		require.NoError(t, err)
		assert.EqualT(t, 0.0, got.Q)
	})

	t.Run("malformed q ignored, default 1.0", func(t *testing.T) {
		got, err := Parse("text/plain;q=garbage")
		require.NoError(t, err)
		assert.EqualT(t, 1.0, got.Q)
	})
}

func TestString(t *testing.T) {
	cases := []struct {
		in   MediaType
		want string
	}{
		{MediaType{Type: tApp, Subtype: tJSON}, jsonMime},
		{MediaType{Type: "text", Subtype: "plain", Params: map[string]string{pCharset: vUTF8}}, "text/plain; charset=utf-8"},
		{MediaType{Type: "*", Subtype: "*"}, starStar},
		{MediaType{}, ""},
	}
	for _, c := range cases {
		t.Run(c.want, func(t *testing.T) {
			assert.EqualT(t, c.want, c.in.String())
		})
	}
}

func TestRoundtrip(t *testing.T) {
	// String() must be re-parseable to an equivalent value.
	inputs := []string{
		jsonMime,
		textPlainUTF8,
		mtAPIJSON,
		"multipart/form-data;boundary=xyz",
	}
	for _, s := range inputs {
		t.Run(s, func(t *testing.T) {
			m1, err := Parse(s)
			require.NoError(t, err)
			s2 := m1.String()
			m2, err := Parse(s2)
			require.NoError(t, err)
			assert.EqualT(t, m1.Type, m2.Type)
			assert.EqualT(t, m1.Subtype, m2.Subtype)
			assert.EqualT(t, m1.Suffix, m2.Suffix)
			assert.EqualValues(t, m1.Params, m2.Params)
		})
	}
}

func TestSpecificity(t *testing.T) {
	cases := []struct {
		s    string
		want int
	}{
		{starStar, 0},
		{textWild, 1},
		{textPlain, 2},
		{textPlainUTF8, 3},
	}
	for _, c := range cases {
		t.Run(c.s, func(t *testing.T) {
			m, err := Parse(c.s)
			require.NoError(t, err)
			assert.EqualT(t, c.want, m.Specificity())
		})
	}
}

func TestMatches(t *testing.T) {
	cases := []struct {
		name      string
		bound     string // receiver
		other     string // argument
		wantMatch bool
	}{
		// type-level
		{"identical bare", textPlain, textPlain, true},
		{"different bare", textPlain, textHTML, false},
		{"different top-level type", textPlain, jsonMime, false},

		// wildcards on the bound
		{"bound */*", starStar, "anything/whatever", true},
		{"bound type/*", textWild, textPlain, true},
		{"bound type/* mismatched type", textWild, imagePNG, false},

		// wildcards on the constraint
		{"other */*", textPlain, starStar, true},
		{"other type/*", textPlain, textWild, true},
		{"other type/* mismatched type", textPlain, imageWild, false},

		// param subset rule (#136)
		{"bare bound, params other → accept", textPlain, textPlainUTF8, true},
		{"bound has params, bare other → accept (no constraint)", textPlainUTF8, textPlain, true},
		{"exact param match", textPlainUTF8, textPlainUTF8, true},
		{"value differs → reject", textPlainUTF8, "text/plain;charset=ascii", false},
		{"key not in bound → reject", textPlainUTF8, "text/plain;version=2", false},
		{"value compare case-insensitive", textPlainUTF8, "text/plain;charset=UTF-8", true},
		{"key compare lowercased at parse", "text/plain;CHARSET=utf-8", textPlainUTF8, true},
		{"bound has extra param, other subset → accept",
			"text/plain;charset=utf-8;boundary=xyz",
			textPlainUTF8, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b, err := Parse(c.bound)
			require.NoError(t, err)
			o, err := Parse(c.other)
			require.NoError(t, err)
			assert.EqualT(t, c.wantMatch, b.Matches(o), "%q.Matches(%q)", c.bound, c.other)
		})
	}
}

func TestParseAccept(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		got := ParseAccept("")
		assert.Nil(t, got)
	})

	t.Run("single", func(t *testing.T) {
		got := ParseAccept(jsonMime)
		require.Len(t, got, 1)
		assert.EqualT(t, tApp, got[0].Type)
		assert.EqualT(t, tJSON, got[0].Subtype)
	})

	t.Run("multiple with q-values", func(t *testing.T) {
		got := ParseAccept("text/html;q=0.8, application/json, */*;q=0.1")
		require.Len(t, got, 3)
		assert.EqualT(t, 0.8, got[0].Q)
		assert.EqualT(t, 1.0, got[1].Q)
		assert.EqualT(t, 0.1, got[2].Q)
	})

	t.Run("malformed entries skipped", func(t *testing.T) {
		got := ParseAccept("application/json, garbage(, text/plain")
		require.Len(t, got, 2)
		assert.EqualT(t, tJSON, got[0].Subtype)
		assert.EqualT(t, tPlain, got[1].Subtype)
	})

	t.Run("quoted comma not split", func(t *testing.T) {
		got := ParseAccept(`text/plain;foo="a,b", text/html`)
		require.Len(t, got, 2)
	})
}

func TestBestMatch(t *testing.T) {
	type row struct {
		name     string
		accept   string
		offered  []string
		wantBest string // empty means no match
	}

	// All rows below are reproductions of the legacy negotiate_test.go
	// cases (see middleware/negotiate_test.go), confirming the new
	// algorithm yields identical answers under default behaviour for
	// inputs that have no parameters or have parameters where both
	// sides agree. Cases where parameters disagree (the A.4 fix) are
	// covered separately in the negotiate_test.go matrix.
	rows := []row{
		{"reject all via q=0", "text/html, */*;q=0", []string{xy}, ""},
		{"wildcard catches anything", "text/html, */*", []string{xy}, xy},
		{"first offer wins on tie", "text/html, image/png", []string{textHTML, imagePNG}, textHTML},
		{"first offer wins on tie (reversed)", "text/html, image/png", []string{imagePNG, textHTML}, imagePNG},
		{"accept earlier specific beats generic", htmlPNGq05, []string{imagePNG}, imagePNG},
		{"q wins over position", htmlPNGq05, []string{textHTML}, textHTML},
		{"no offer matches", htmlPNGq05, []string{"foo/bar"}, ""},
		{"higher q wins", htmlPNGq05, []string{imagePNG, textHTML}, textHTML},
		{"higher q wins even when offer lists png first", htmlPNGq05, []string{textHTML, imagePNG}, textHTML},
		{"higher q overrides offer order", html05PNG, []string{imagePNG}, imagePNG},
		{"higher q overrides offer order 2", html05PNG, []string{textHTML}, textHTML},
		{"higher q image/png beats text/html;q=0.5", html05PNG, []string{imagePNG, textHTML}, imagePNG},
		{"text/html;q=0.5, image/png with both offers", html05PNG, []string{textHTML, imagePNG}, imagePNG},
		{"image/png beats image/* on specificity", pngWildq05, []string{imageJPG, imagePNG}, imagePNG},
		{"image/* matches jpg", pngWildq05, []string{imageJPG}, imageJPG},
		{"image/* matches first jpg, jpg before gif", pngWildq05, []string{imageJPG, imageGIF}, imageJPG},
		{"image/* matches both, first wins", pngWild, []string{imageJPG, imageGIF}, imageJPG},
		{"image/* matches gif first", pngWild, []string{imageGIF, imageJPG}, imageGIF},
		{"image/png beats image/* on specificity (2)", pngWild, []string{imageGIF, imagePNG}, imagePNG},
		{"image/png beats image/* (offer order doesn't override)", pngWild, []string{imagePNG, imageGIF}, imagePNG},
		{"vendor params don't break match", "application/vnd.google.protobuf;proto=io.prometheus.client.MetricFamily;encoding=delimited;q=0.7,text/plain;version=0.0.4;q=0.3", []string{textPlain}, textPlain},
		// vendor MIME types are NOT structurally matched against
		// "+json" — text/json doesn't match application/vnd.cia.v1+json.
		{"vendor MIME unmatched", jsonMime, []string{"application/vnd.cia.v1+json"}, ""},
		// java client default
		{"java default", "text/html, image/gif, image/jpeg, *; q=.2, */*; q=.2", []string{jsonMime}, jsonMime},
	}

	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			accept := ParseAccept(r.accept)
			offers := make(Set, 0, len(r.offered))
			for _, o := range r.offered {
				m, err := Parse(o)
				require.NoErrorf(t, err, "offer %q", o)
				offers = append(offers, m)
			}
			best, ok := accept.BestMatch(offers)
			if r.wantBest == "" {
				assert.FalseT(t, ok, "want no match, got %q", best.String())
				return
			}
			require.TrueT(t, ok)
			assert.EqualT(t, r.wantBest, best.String())
		})
	}
}

func TestBestMatchEmptyInputs(t *testing.T) {
	t.Run("empty accept set", func(t *testing.T) {
		offers := Set{mustParse(t, textPlain)}
		_, ok := Set(nil).BestMatch(offers)
		assert.FalseT(t, ok)
	})

	t.Run("empty offered set", func(t *testing.T) {
		accept := ParseAccept(textPlain)
		_, ok := accept.BestMatch(nil)
		assert.FalseT(t, ok)
	})
}

func TestStripParams(t *testing.T) {
	m := mustParse(t, "text/plain;charset=utf-8;q=0.5")
	stripped := m.StripParams()
	assert.Nil(t, stripped.Params)
	assert.EqualT(t, "text", stripped.Type)
	assert.EqualT(t, "plain", stripped.Subtype)
	// q is preserved — it is meta and still drives negotiation order.
	assert.EqualT(t, 0.5, stripped.Q)
	// original is untouched
	require.NotNil(t, m.Params)
}

func mustParse(t *testing.T, s string) MediaType {
	t.Helper()
	m, err := Parse(s)
	require.NoError(t, err)
	return m
}
