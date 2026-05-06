// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Copyright 2013 The Go Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd.

// this file was originally based on github.com/golang/gddo

package middleware

import (
	"net/http"
	"strings"

	"github.com/go-openapi/runtime/middleware/header"
	"github.com/go-openapi/runtime/server-middleware/mediatype"
)

// NegotiateOption configures [NegotiateContentType] behaviour.
type NegotiateOption func(*negotiateOptions)

type negotiateOptions struct {
	ignoreParameters bool
}

func negotiateOptionsWithDefaults(opts []NegotiateOption) negotiateOptions {
	var o negotiateOptions
	for _, apply := range opts {
		apply(&o)
	}

	return o
}

// WithIgnoreParameters returns a [NegotiateOption] that strips MIME-type
// parameters from both Accept entries and offers before matching, restoring
// the behaviour the runtime had before v0.30.
//
// New code should leave parameters honoured (the default). This option
// exists for applications that depend on the looser pre-v0.30 match —
// most often because their producers and Accept clients use mismatched
// charset or version params that they treat as informational.
//
// Example — per-call opt-out:
//
//	chosen := middleware.NegotiateContentType(r, offers, "",
//	    middleware.WithIgnoreParameters(true),
//	)
//
// Example — server-wide opt-out:
//
//	ctx := middleware.NewContext(spec, api, nil).SetIgnoreParameters(true)
func WithIgnoreParameters(ignore bool) NegotiateOption {
	return func(o *negotiateOptions) {
		o.ignoreParameters = ignore
	}
}

// NegotiateContentEncoding returns the best offered content encoding for the
// request's Accept-Encoding header. If two offers match with equal weight and
// then the offer earlier in the list is preferred. If no offers are
// acceptable, then "" is returned.
//
// Encoding tokens have no parameters, so this function is unaffected by
// the v0.30 parameter-honouring change to [NegotiateContentType].
func NegotiateContentEncoding(r *http.Request, offers []string) string {
	bestOffer := "identity"
	bestQ := -1.0
	specs := header.ParseAccept(r.Header, "Accept-Encoding")
	for _, offer := range offers {
		for _, spec := range specs {
			if spec.Q > bestQ &&
				(spec.Value == "*" || spec.Value == offer) {
				bestQ = spec.Q
				bestOffer = offer
			}
		}
	}
	if bestQ == 0 {
		bestOffer = ""
	}

	return bestOffer
}

// NegotiateContentType returns the best offered content type for the
// request's Accept header. If two offers match with equal weight, then
// the more specific offer is preferred (text/* trumps */*; type/subtype
// trumps type/*). If two offers match with equal weight and specificity,
// then the offer earlier in the list is preferred. If no offers match,
// then defaultOffer is returned.
//
// As of v0.30 the matching rule honours MIME-type parameters: an Accept
// entry of "text/plain;charset=utf-8" matches an offer of bare
// "text/plain" (offer carries no constraint), but it does NOT match an
// offer of "text/plain;charset=ascii" (charset values disagree). Pass
// [WithIgnoreParameters](true) to restore the pre-v0.30 behaviour where
// parameters were stripped before matching — see [WithIgnoreParameters]
// for details and an example.
//
// When the Accept header is absent, the first offer is returned
// unchanged (param-stripping is irrelevant in that case).
func NegotiateContentType(r *http.Request, offers []string, defaultOffer string, opts ...NegotiateOption) string {
	if len(offers) == 0 {
		return defaultOffer
	}
	o := negotiateOptionsWithDefaults(opts)

	// Per RFC 7230 §3.2.2, multiple Accept headers are equivalent to a
	// single comma-joined value. Join before parsing so we don't drop
	// later entries.
	acceptValues := r.Header.Values("Accept")
	if len(acceptValues) == 0 {
		return offers[0]
	}
	acceptSet := mediatype.ParseAccept(strings.Join(acceptValues, ", "))
	if len(acceptSet) == 0 {
		return defaultOffer
	}

	offerSet := make(mediatype.Set, 0, len(offers))
	rawByIdx := make([]string, 0, len(offers))
	for _, raw := range offers {
		mt, err := mediatype.Parse(raw)
		if err != nil {
			continue
		}
		offerSet = append(offerSet, mt)
		rawByIdx = append(rawByIdx, raw)
	}
	if len(offerSet) == 0 {
		return defaultOffer
	}

	if o.ignoreParameters {
		acceptSet = stripSet(acceptSet)
		offerSet = stripSet(offerSet)
	}

	best, ok := acceptSet.BestMatch(offerSet)
	if !ok {
		return defaultOffer
	}
	// Return the original raw offer string so callers receive the value
	// they declared, with its parameters preserved.
	for i, mt := range offerSet {
		if mt.Type == best.Type && mt.Subtype == best.Subtype && sameParams(mt.Params, best.Params) {
			return rawByIdx[i]
		}
	}
	return best.String()
}

func stripSet(s mediatype.Set) mediatype.Set {
	out := make(mediatype.Set, len(s))
	for i, m := range s {
		out[i] = m.StripParams()
	}

	return out
}

func sameParams(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}

	return true
}

func normalizeOffers(orig []string) (norm []string) {
	for _, o := range orig {
		norm = append(norm, normalizeOffer(o))
	}
	return
}

func normalizeOffer(orig string) string {
	const maxParts = 2
	return strings.SplitN(orig, ";", maxParts)[0]
}
