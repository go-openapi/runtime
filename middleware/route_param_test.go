// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
)

func TestRouteParams(t *testing.T) {
	coll1 := RouteParams([]RouteParam{
		{"blah", "foo"},
		{"abc", "bar"},
		{"ccc", "efg"},
	})

	v := coll1.Get("blah")
	assert.Equal(t, "foo", v)
	v2 := coll1.Get("abc")
	assert.Equal(t, "bar", v2)
	v3 := coll1.Get("ccc")
	assert.Equal(t, "efg", v3)
	v4 := coll1.Get("ydkdk")
	assert.Empty(t, v4)
}
