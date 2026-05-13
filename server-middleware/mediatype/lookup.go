// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package mediatype

// Lookup finds the entry in m matching mediaType, with alias-aware
// fallback. It is the canonical seam for codec-map lookups in both
// the client and server runtimes — placing the fallback policy here
// keeps alias definitions (and any future lookup tolerances) in one
// place.
//
// Lookup tries the following, in order, returning the first hit:
//
//  1. mediaType verbatim (fast path for callers that already pass a
//     canonical, parameter-free string and store map keys in the
//     same form).
//  2. The canonical "type/subtype" form derived by parsing
//     mediaType (strips parameters and lowercases — recovers the
//     match when mediaType carries "; charset=...").
//  3. The alias-canonicalized form from the package-internal alias
//     table — for example, a request for "application/yaml" finds
//     an entry registered under "application/x-yaml".
//  4. Walks m and returns the first entry whose own key
//     alias-canonicalizes to the same target as mediaType. This
//     covers the "map keyed by one alias, query uses another alias
//     of the same canonical" case (e.g. registered under text/yaml,
//     queried as application/x-yaml).
//  5. When [AllowSuffix] is passed in opts: the RFC 6839
//     structured-syntax suffix base on the query side (plus its own
//     alias canonical). Catches the "spec/traffic divergence" case
//     (request for application/vnd.api+json finds a JSON consumer
//     registered under application/json). Query-side only — no
//     map-side suffix folding.
//
// Lookup does NOT fall back to "*/*". Callers that want wildcard
// behavior (the historical resolveConsumer pattern in the client
// runtime) chain that themselves after a Lookup miss — keeping
// wildcard semantics explicit at each call site.
//
// Map keys are expected in canonical "type/subtype" form (no
// parameters). The runtime's default Consumers / Producers maps
// follow this convention.
//
// Returns (zero, false) when:
//
//   - m is empty;
//   - mediaType fails to parse and is not present verbatim;
//   - none of the active tiers hits.
//
// The malformed-vs-not-found distinction is intentionally elided:
// codec-lookup callers treat both as the same "no codec" error path.
func Lookup[T any](m map[string]T, mediaType string, opts ...MatchOption) (T, bool) {
	var zero T
	if len(m) == 0 {
		return zero, false
	}
	o := applyMatchOptions(opts)
	// Tier 1: raw key.
	if v, ok := m[mediaType]; ok {
		return v, true
	}
	mt, err := Parse(mediaType)
	if err != nil {
		return zero, false
	}
	key := mt.Type + "/" + mt.Subtype
	// Tier 2: parsed canonical form (strips params, lowercases).
	if key != mediaType {
		if v, ok := m[key]; ok {
			return v, true
		}
	}
	// Tier 3: alias bridge from the query side — fast path when the
	// map is keyed by the canonical form.
	target := key
	if canon, ok := aliases[key]; ok {
		target = canon
		if v, ok := m[target]; ok {
			return v, true
		}
	}
	// Tier 4: alias bridge from the map side. Catches the case where
	// the map is keyed by an alias (or by a different alias than the
	// query). O(len(m)) but codec maps are tiny (<10 entries) so this
	// is negligible.
	for k, v := range m {
		kCanon := k
		if c, ok := aliases[k]; ok {
			kCanon = c
		}
		if kCanon == target {
			return v, true
		}
	}
	// Tier 5 (opt-in): RFC 6839 structured-syntax suffix base. Only
	// fires when the query carries a known suffix and the suffix
	// resolves to a base type in the package-internal suffix→base
	// table. Query-side suffix fold only — we still walk the map
	// applying alias canonicalization (same as tier 4), but never
	// fold a map key's suffix.
	if o.allowSuffix && mt.Suffix != "" {
		base := mt.Base()
		if base.Type != mt.Type || base.Subtype != mt.Subtype {
			baseKey := base.Type + "/" + base.Subtype
			baseTarget := baseKey
			if v, ok := m[baseKey]; ok {
				return v, true
			}
			if canon, ok := aliases[baseKey]; ok {
				baseTarget = canon
				if v, ok := m[canon]; ok {
					return v, true
				}
			}
			// Catches the "map keyed by an alias of the suffix base"
			// case (e.g. base resolves to application/yaml, map is
			// keyed by application/x-yaml).
			for k, v := range m {
				kCanon := k
				if c, ok := aliases[k]; ok {
					kCanon = c
				}
				if kCanon == baseTarget {
					return v, true
				}
			}
		}
	}
	return zero, false
}
