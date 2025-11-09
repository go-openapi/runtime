// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthInfoWriter(t *testing.T) {
	const bearerToken = "Bearer the-token-goes-here"

	hand := ClientAuthInfoWriterFunc(func(r ClientRequest, _ strfmt.Registry) error {
		return r.SetHeaderParam(HeaderAuthorization, bearerToken)
	})

	tr := new(TestClientRequest)
	require.NoError(t, hand.AuthenticateRequest(tr, nil))
	assert.Equal(t, bearerToken, tr.Headers.Get(HeaderAuthorization))
}
