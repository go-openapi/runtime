// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package opentracing

import (
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/client"
	opentracing "github.com/opentracing/opentracing-go"
)

// WithOpenTracing adds opentracing support to the provided runtime.
// A new client span is created for each request.
//
// If the context of the client operation does not contain an active span, no span is created.
// The provided opts are applied to each spans - for example to add global tags.
//
// This method is provided to continue supporting users of [github.com/go-openapi/runtime] who
// still rely on opentracing and have not been able to transition to opentelemetry yet.
func WithOpenTracing(r *client.Runtime, opts ...opentracing.StartSpanOption) runtime.ClientTransport {
	return newOpenTracingTransport(r, r.Host, opts)
}
