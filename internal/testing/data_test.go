// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package testing

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInvalidJSON(t *testing.T) {
	require.NotEmpty(t, InvalidJSONMessage)
}
