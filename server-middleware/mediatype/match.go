// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package mediatype

// MatchFirst reports whether actual matches any entry in allowed, using the
// param-aware rule from [MediaType.Matches].
//
// MatchFirst short-circuits on the first allowed entry that accepts actual:
// the returned [MediaType] may not be the most specific match. Callers that
// need ranked matching should use [Set.BestMatch].
//
// Return values:
//
//   - (matched, true,  nil)        — the first allowed entry that matches.
//   - (zero,    false, nil)        — actual is well-formed but no allowed
//     entry accepts it.
//     Maps to an HTTP 415 outcome.
//   - (zero,    false, err)        — actual fails to parse. err wraps
//     [ErrMalformed], so callers can use [errors.Is] to distinguish this
//     case.
//     Maps to an HTTP 400 outcome.
//
// Allowed entries that themselves fail to parse are skipped (they cannot
// match any well-formed actual), and no error is surfaced for them.
//
// An empty allowed list returns (zero, false, nil). MatchFirst is the primitive;
// callers decide what no-constraints means in their context.
func MatchFirst(allowed []string, actual string) (MediaType, bool, error) {
	if len(allowed) == 0 {
		return MediaType{}, false, nil
	}
	actualMT, err := Parse(actual)
	if err != nil {
		return MediaType{}, false, err
	}
	for _, a := range allowed {
		allowedMT, perr := Parse(a)
		if perr != nil {
			continue
		}
		if allowedMT.Matches(actualMT) {
			return allowedMT, true, nil
		}
	}

	return MediaType{}, false, nil
}
