// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"github.com/go-openapi/runtime/server-middleware/docui"
	"github.com/go-openapi/runtime/server-middleware/negotiate"
)

// NegotiateOption configures [NegotiateContentType] behaviour.
//
// Deprecated: moved to the [negotiate] package. Use [negotiate.Option] instead.
type NegotiateOption = negotiate.Option

// NegotiateContentType returns the best offered content type for the
// request's Accept header.
//
// Deprecated: moved to the [negotiate] package. Use [negotiate.ContentType] instead.
var NegotiateContentType = negotiate.ContentType

// NegotiateContentEncoding returns the best offered content encoding for
// the request's Accept-Encoding header.
//
// Deprecated: moved to the [negotiate] package. Use [negotiate.ContentEncoding] instead.
var NegotiateContentEncoding = negotiate.ContentEncoding

// WithIgnoreParameters returns a [NegotiateOption] that strips MIME-type
// parameters from both Accept entries and offers before matching,
// restoring the pre-v0.30 behaviour.
//
// Deprecated: moved to the [negotiate] package. Use [negotiate.WithIgnoreParameters] instead.
var WithIgnoreParameters = negotiate.WithIgnoreParameters

// RapiDocOpts configures the [RapiDoc] middlewares.
//
// Deprecated: moved to the [docui] package. Use [docui.RapiDocOpts] instead.
type RapiDocOpts = docui.RapiDocOpts

// RapiDoc creates a [middleware] to serve a documentation site for a swagger spec.
//
// This allows for altering the spec before starting the [http] listener.
//
// Deprecated: moved to the [docui] package. Use [docui.RapiDoc] instead.
var RapiDoc = docui.RapiDoc

// RedocOpts configures the [Redoc] middlewares.
//
// Deprecated: moved to the [docui] package. Use [docui.RedocOpts] instead.
type RedocOpts = docui.RedocOpts

// Redoc creates a [middleware] to serve a documentation site for a swagger spec.
//
// This allows for altering the spec before starting the [http] listener.
//
// Deprecated: moved to the [docui] package. Use [docui.Redoc] instead.
var Redoc = docui.Redoc

// SwaggerUIOpts configures the [SwaggerUI] [middleware].
//
// Deprecated: moved to the [docui] package. Use [docui.SwaggerUIOpts] instead.
type SwaggerUIOpts = docui.SwaggerUIOpts

// SwaggerUI creates a [middleware] to serve a documentation site for a swagger spec.
//
// This allows for altering the spec before starting the [http] listener.
//
// Deprecated: moved to the [docui] package. Use [docui.SwaggerUI] instead.
var SwaggerUI = docui.SwaggerUI

// SwaggerUIOAuth2Callback creates a middleware that serves the OAuth2 callback page used by Swagger UI.
//
// Deprecated: moved to the [docui] package. Use [docui.SwaggerUIOAuth2Callback] instead.
var SwaggerUIOAuth2Callback = docui.SwaggerUIOAuth2Callback

// SpecOption can be applied to the [Spec] serving [middleware].
//
// Deprecated: moved to the [docui] package. Use [docui.SpecOption] instead.
type SpecOption = docui.SpecOption

// Spec creates a [middleware] to serve a swagger spec as a JSON document.
//
// This allows for altering the spec before starting the [http] listener.
//
// The basePath argument indicates the path of the spec document (defaults to "/").
// Additional [SpecOption] can be used to change the name of the document (defaults to "swagger.json").
//
// Deprecated: moved to the [docui] package as [docui.ServeSpec].
var Spec = docui.ServeSpec

// WithSpecPath sets the path to be joined to the base path of the [Spec] [middleware].
//
// This is empty by default.
//
// Deprecated: moved to the [docui] package. Use [docui.WithSpecPath] instead.
var WithSpecPath = docui.WithSpecPath

// WithSpecDocument sets the name of the JSON document served as a spec.
//
// By default, this is "swagger.json".
//
// Deprecated: moved to the [docui] package. Use [docui.WithSpecDocument] instead.
var WithSpecDocument = docui.WithSpecDocument

// UIOption can be applied to UI serving [middleware], such as Context.[APIHandler] or
// Context.[APIHandlerSwaggerUI] to alter the default behavior.
//
// Deprecated: moved to the [docui] package. Use [docui.UIOption] instead.
type UIOption = docui.UIOption

// WithUIBasePath sets the base path from where to serve the UI assets.
//
// By default, Context [middleware] sets this value to the API base path.
//
// Deprecated: moved to the [docui] package. Use [docui.WithUIBasePath] instead.
var WithUIBasePath = docui.WithUIBasePath

// WithUIPath sets the path from where to serve the UI assets (i.e. /{basepath}/{path}.
//
// Deprecated: moved to the [docui] package. Use [docui.WithUIPath] instead.
var WithUIPath = docui.WithUIPath

// WithUISpecURL sets the path from where to serve swagger spec document.
//
// This may be specified as a full URL or a path.
//
// By default, this is "/swagger.json".
//
// Deprecated: moved to the [docui] package. Use [docui.WithUISpecURL] instead.
var WithUISpecURL = docui.WithUISpecURL

// WithUITitle sets the title of the UI.
//
// By default, Context [middleware] sets this value to the title found in the API spec.
//
// Deprecated: moved to the [docui] package. Use [docui.WithUITitle] instead.
var WithUITitle = docui.WithUITitle

// WithTemplate allows to set a custom template for the UI.
//
// UI [middleware] will panic if the template does not parse or execute properly.
//
// Deprecated: moved to the [docui] package. Use [docui.WithTemplate] instead.
var WithTemplate = docui.WithTemplate
