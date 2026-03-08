// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"testing"

	"github.com/go-openapi/testify/v2/require"
)

func TestGetOK(t *testing.T) {
	m := make(map[string][]string)
	m["key1"] = []string{"value1"}
	m["key2"] = []string{}
	values := Values(m)

	v, hasKey, hasValue := values.GetOK("key1")
	require.Equal(t, []string{"value1"}, v)
	require.TrueT(t, hasKey)
	require.TrueT(t, hasValue)

	v, hasKey, hasValue = values.GetOK("key2")
	require.Equal(t, []string{}, v)
	require.TrueT(t, hasKey)
	require.FalseT(t, hasValue)

	v, hasKey, hasValue = values.GetOK("key3")
	require.Nil(t, v)
	require.FalseT(t, hasKey)
	require.FalseT(t, hasValue)
}
