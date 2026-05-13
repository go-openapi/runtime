// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package mediatype

// MatchFirst reports whether actual matches any entry in allowed,
// using [MediaType.Match] — the param-aware RFC 7231 rule plus the
// alias bridge from the package-internal alias table.
//
// The scan is two-pass: the first pass returns the first allowed
// entry that matches under [MatchExact] (RFC 7231 semantics); only
// when no exact match is found does the second pass look for a
// [MatchAlias] match. This preserves the "exact beats alias" tier
// from [Set.BestMatch] while keeping the "first match wins"
// semantics within each tier.
//
// Return values:
//
//   - (matched, true,  nil)        — the first allowed entry that
//     matches, with exact matches preferred over alias matches.
//   - (zero,    false, nil)        — actual is well-formed but no
//     allowed entry accepts it. Maps to an HTTP 415 outcome.
//   - (zero,    false, err)        — actual fails to parse. err
//     wraps [ErrMalformed], so callers can use [errors.Is] to
//     distinguish this case. Maps to an HTTP 400 outcome.
//
// Allowed entries that themselves fail to parse are skipped (they
// cannot match any well-formed actual), and no error is surfaced
// for them.
//
// An empty allowed list returns (zero, false, nil). MatchFirst is
// the primitive; callers decide what no-constraints means in their
// context.
func MatchFirst(allowed []string, actual string) (MediaType, bool, error) {
	if len(allowed) == 0 {
		return MediaType{}, false, nil
	}
	actualMT, err := Parse(actual)
	if err != nil {
		return MediaType{}, false, err
	}
	// Two-pass scan: exact tier first, alias tier as fallback. The
	// allowed list is typically short (an operation's Consumes set),
	// so re-parsing each entry on the alias pass is cheaper than
	// caching parses across both.
	for _, a := range allowed {
		allowedMT, perr := Parse(a)
		if perr != nil {
			continue
		}
		if allowedMT.Match(actualMT) == MatchExact {
			return allowedMT, true, nil
		}
	}
	for _, a := range allowed {
		allowedMT, perr := Parse(a)
		if perr != nil {
			continue
		}
		if allowedMT.Match(actualMT) == MatchAlias {
			return allowedMT, true, nil
		}
	}

	return MediaType{}, false, nil
}
