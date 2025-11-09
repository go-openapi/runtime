// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package petstore

import (
	"testing"

	"github.com/go-openapi/testify/v2/require"
)

func TestAPI(t *testing.T) {
	doc, api := NewAPI(t)

	require.NotNil(t, doc)
	require.NotNil(t, api)
}
