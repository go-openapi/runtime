// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestBindMapWithNilSchema verifies that binding into a map[string]any
// does not panic when a parameter has a nil Schema.
//
// Regression test for https://github.com/go-openapi/runtime/issues/487.
func TestBindMapWithNilSchema(t *testing.T) {
	// Build a parameter whose Schema is nil (e.g. a non-body param with
	// a type that the binder cannot resolve to a reflect.Type).
	p := spec.QueryParam(paramKeyName).Typed(typeString, "")

	params := map[string]spec.Parameter{
		keyName: *p,
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, testURL+"?name=alice", nil)
	require.NoError(t, err)

	binder := NewUntypedRequestBinder(params, new(spec.Swagger), strfmt.Default)

	data := make(map[string]any)
	err = binder.Bind(req, nil, runtime.JSONConsumer(), &data)
	require.NoError(t, err)

	assert.Equal(t, "alice", data[paramKeyName])
}

// TestBindMapWithNilSchema_ArrayType verifies the array-schema path when
// binding into a map and Schema is non-nil with type=array.
func TestBindMapWithNilSchema_ArrayType(t *testing.T) {
	arraySchema := new(spec.Schema).Typed(typeArray, "")
	p := spec.BodyParam(keyFriend, arraySchema)

	params := map[string]spec.Parameter{
		keyFriend: *p,
	}

	body := []byte(`["a","b"]`)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, testURL, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", jsonMime)

	binder := NewUntypedRequestBinder(params, new(spec.Swagger), strfmt.Default)

	data := make(map[string]any)
	err = binder.Bind(req, nil, runtime.JSONConsumer(), &data)
	require.NoError(t, err)
}

// FuzzUntypedRequestBinder exercises UntypedRequestBinder.Bind with
// fuzz-generated parameter definitions bound into a map[string]any.
//
// The main goal is to verify that no combination of parameter attributes
// causes a panic (e.g. nil pointer dereference on param.Schema).
func FuzzUntypedRequestBinder(f *testing.F) {
	// Seed corpus: representative parameter shapes.
	f.Add("name", "query", "string", "", false, false)
	f.Add("id", "path", "integer", "int64", false, false)
	f.Add("tags", "query", "array", "csv", true, false)
	f.Add("body", "body", "object", "", false, true)
	f.Add("flag", "query", "boolean", "", false, false)
	f.Add("score", "query", "number", "double", false, false)
	f.Add("", "query", "", "", false, false)
	f.Add("x", "header", "string", "", false, false)

	f.Fuzz(func(_ *testing.T, name, in, tpe, format string, isArray, hasSchema bool) {
		if name == "" {
			name = "p"
		}

		var param spec.Parameter
		switch in {
		case "query":
			param = *spec.QueryParam(name)
		case "path":
			param = *spec.PathParam(name)
		case "header":
			param = *spec.HeaderParam(name)
		case "body":
			if hasSchema {
				schema := new(spec.Schema).Typed(tpe, format)
				param = *spec.BodyParam(name, schema)
			} else {
				// Body param with nil schema — edge case from #487.
				param = spec.Parameter{}
				param.Name = name
				param.In = "body"
			}
		default:
			param = *spec.QueryParam(name)
		}

		// For non-body params, set Type directly.
		if in != "body" {
			param.Type = tpe
			param.Format = format

			if isArray {
				param.Type = typeArray
				items := new(spec.Items)
				items.Type = typeString
				param.Items = items
				param.CollectionFormat = "csv"
			}
		}

		params := map[string]spec.Parameter{
			"P": param,
		}

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, testURL, nil)
		if err != nil {
			return
		}

		binder := NewUntypedRequestBinder(params, new(spec.Swagger), strfmt.Default)

		// Bind into a map — this exercises the isMap==true path where
		// the nil Schema dereference occurred.
		data := make(map[string]any)
		_ = binder.Bind(req, nil, runtime.JSONConsumer(), &data) // must not panic
	})
}
