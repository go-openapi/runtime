// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package docui

import (
	"bytes"
	"encoding/gob"
	"strings"
)

const (
	// constants that are common to all UI-serving middlewares.
	defaultDocsPath  = "docs"
	defaultDocsURL   = "/swagger.json"
	defaultDocsTitle = "API Documentation"

	contentTypeHeader = "Content-Type"
	applicationJSON   = "application/json"
)

// UIOptions defines common options for UI serving middlewares.
type UIOptions struct {
	// BasePath for the UI, defaults to: /
	BasePath string

	// Path combines with BasePath to construct the path to the UI, defaults to: "docs".
	Path string

	// SpecURL is the URL of the spec document.
	//
	// Defaults to: /swagger.json
	SpecURL string

	// Title for the documentation site, default to: API documentation
	Title string

	// Template specifies a custom template to serve the UI
	Template string
}

// ToCommonUIOptions converts any UI option type to retain the common options.
//
// This uses gob encoding/decoding to convert common fields from one struct to another.
func ToCommonUIOptions(opts any) UIOptions {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	dec := gob.NewDecoder(&buf)
	var o UIOptions
	err := enc.Encode(opts)
	if err != nil {
		panic(err)
	}

	err = dec.Decode(&o)
	if err != nil {
		panic(err)
	}

	return o
}

// FromCommonToAnyOptions copies the common UI options held in source into the
// flavor-specific target struct (one of [SwaggerUIOpts], [RedocOpts] or
// [RapiDocOpts]).
func FromCommonToAnyOptions[T any](source UIOptions, target *T) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	dec := gob.NewDecoder(&buf)
	err := enc.Encode(source)
	if err != nil {
		panic(err)
	}

	err = dec.Decode(target)
	if err != nil {
		panic(err)
	}
}

// UIOption can be applied to UI serving [middleware] to alter the default
// behavior.
type UIOption func(*UIOptions)

// UIOptionsWithDefaults applies the given options on top of an empty
// [UIOptions]. Per-flavor handlers ([SwaggerUI], [Redoc], [RapiDoc])
// fill in the remaining defaults via [UIOptions.EnsureDefaults] when
// the option struct is used.
func UIOptionsWithDefaults(opts []UIOption) UIOptions {
	var o UIOptions
	for _, apply := range opts {
		apply(&o)
	}

	return o
}

// WithUIBasePath sets the base path from where to serve the UI assets.
func WithUIBasePath(base string) UIOption {
	return func(o *UIOptions) {
		if !strings.HasPrefix(base, "/") {
			base = "/" + base
		}
		o.BasePath = base
	}
}

// WithUIPath sets the path from where to serve the UI assets (i.e. /{basepath}/{path}.
func WithUIPath(pth string) UIOption {
	return func(o *UIOptions) {
		o.Path = pth
	}
}

// WithUISpecURL sets the path from where to serve swagger spec document.
//
// This may be specified as a full URL or a path.
//
// By default, this is "/swagger.json".
func WithUISpecURL(specURL string) UIOption {
	return func(o *UIOptions) {
		o.SpecURL = specURL
	}
}

// WithUITitle sets the title of the UI.
func WithUITitle(title string) UIOption {
	return func(o *UIOptions) {
		o.Title = title
	}
}

// WithTemplate allows to set a custom template for the UI.
//
// UI [middleware] will panic if the template does not parse or execute properly.
func WithTemplate(tpl string) UIOption {
	return func(o *UIOptions) {
		o.Template = tpl
	}
}

// EnsureDefaults in case some options are missing.
func (r *UIOptions) EnsureDefaults() {
	if r.BasePath == "" {
		r.BasePath = "/"
	}
	if r.Path == "" {
		r.Path = defaultDocsPath
	}
	if r.SpecURL == "" {
		r.SpecURL = defaultDocsURL
	}
	if r.Title == "" {
		r.Title = defaultDocsTitle
	}
}
