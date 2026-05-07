// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package mediatype

import (
	"fmt"
	"mime"
	"strconv"
	"strings"
)

const wildcard = "*"

// Specificity scores returned by [MediaType.Specificity], ordered from
// least to most specific.
const (
	SpecificityAny             = iota // "*/*"
	SpecificityType                   // "type/*"
	SpecificityExact                  // "type/subtype" (no params)
	SpecificityExactWithParams        // "type/subtype;k=v"
)

type mediaTypeError string

func (e mediaTypeError) Error() string {
	return string(e)
}

// ErrMalformed is the sentinel returned (wrapped) by [Parse] when its input
// cannot be parsed as an RFC 7231 media type.
//
// Callers can test for it with [errors.Is] to distinguish a client-side
// malformed Content-Type header (an HTTP 400 outcome) from a well-formed
// value that simply matches no allowed entry (an HTTP 415 outcome).
const ErrMalformed mediaTypeError = "mediatype: malformed"

// MediaType is a parsed RFC 7231 media type with optional parameters and
// an optional q-value (used by Accept negotiation).
//
// Type, Subtype and the keys of Params are lowercased. Parameter values
// are preserved verbatim; comparisons are case-insensitive (matching the
// pre-v0.30 behaviour and the common convention for charset, version, etc.).
type MediaType struct {
	Type    string
	Subtype string
	Params  map[string]string
	Q       float64
}

// Parse parses a single media type. The input may carry parameters and a
// q-value; the q-value is extracted into [MediaType.Q] and removed from
// [MediaType.Params].
//
// An empty input returns an error.
func Parse(s string) (MediaType, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return MediaType{}, fmt.Errorf("%w: empty value", ErrMalformed)
	}
	full, params, err := mime.ParseMediaType(s)
	if err != nil {
		return MediaType{}, fmt.Errorf("%w: %w", ErrMalformed, err)
	}
	slash := strings.IndexByte(full, '/')
	if slash <= 0 || slash == len(full)-1 {
		return MediaType{}, fmt.Errorf("%w: %q has no subtype", ErrMalformed, s)
	}
	mt := MediaType{
		Type:    full[:slash],
		Subtype: full[slash+1:],
		Q:       1.0,
	}
	if q, ok := params["q"]; ok {
		if qf, perr := strconv.ParseFloat(q, 64); perr == nil {
			if qf < 0 {
				qf = 0
			}
			if qf > 1 {
				qf = 1
			}
			mt.Q = qf
		}
		delete(params, "q")
	}
	if len(params) > 0 {
		mt.Params = params
	}

	return mt, nil
}

// String renders the canonical "type/subtype;k=v;k=v" form. Parameters are
// emitted in lexicographic key order (the standard library guarantees this)
// so the result is stable. The q-value is NOT emitted — it is meta, not
// part of the media type identity.
func (m MediaType) String() string {
	if m.Type == "" && m.Subtype == "" {
		return ""
	}

	return mime.FormatMediaType(m.Type+"/"+m.Subtype, m.Params)
}

// Matches reports whether the receiver accepts other, per the package
// documentation: the receiver is the bound, other is the constraint.
func (m MediaType) Matches(other MediaType) bool {
	if !typeAgrees(m.Type, other.Type) {
		return false
	}
	if !subtypeAgrees(m.Type, m.Subtype, other.Type, other.Subtype) {
		return false
	}
	if len(m.Params) == 0 {
		return true
	}
	for k, v := range other.Params {
		sv, ok := m.Params[k]
		if !ok || !strings.EqualFold(sv, v) {
			return false
		}
	}

	return true
}

// Specificity returns a numeric score for ordering matches. Higher is more
// specific. The returned value is one of [SpecificityAny],
// [SpecificityType], [SpecificityExact] or [SpecificityExactWithParams].
func (m MediaType) Specificity() int {
	if m.Type == wildcard && m.Subtype == wildcard {
		return SpecificityAny
	}
	if m.Subtype == wildcard {
		return SpecificityType
	}
	if len(m.Params) == 0 {
		return SpecificityExact
	}

	return SpecificityExactWithParams
}

// typeAgrees reports whether two top-level types match, allowing "*" on
// either side. A type of "*" without a "*" subtype is rejected per RFC
// 7231 §5.3.2 ("*/sub" is not valid), but Parse never produces such a
// shape — it goes through mime.ParseMediaType.
func typeAgrees(a, b string) bool {
	return a == wildcard || b == wildcard || a == b
}

// subtypeAgrees handles the "type/*" wildcard: the bare type must match
// (a "*/*" pair has already been accepted by typeAgrees above).
func subtypeAgrees(at, asub, bt, bsub string) bool {
	if at == wildcard || bt == wildcard {
		// at least one side is "*/*" or "*/sub". With typeAgrees having
		// returned true, we accept.
		return true
	}
	if asub == wildcard || bsub == wildcard {
		return true
	}

	return asub == bsub
}

// StripParams returns a copy of m with no parameters. Q is preserved
// because it drives negotiation ordering, not media-type identity.
//
// Useful for the legacy "ignore parameters" negotiation mode.
func (m MediaType) StripParams() MediaType {
	return MediaType{Type: m.Type, Subtype: m.Subtype, Q: m.Q}
}
