// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package request

import (
	"context"
	"errors"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestRecordCredentials(t *testing.T) {
	r := New("GET", "/x", nil)
	require.NoError(t, r.SetHeaderParam("X-Unchanged", "a"))
	require.NoError(t, r.SetHeaderParam("X-Changed", "a"))
	require.NoError(t, r.SetQueryParam("unchanged", "a"))

	err := r.recordCredentials(func() error {
		_ = r.SetHeaderParam("X-Unchanged", "a")
		_ = r.SetHeaderParam("X-Changed", "b")
		_ = r.SetHeaderParam("x-api-key", "k")
		_ = r.SetQueryParam("unchanged", "a")
		_ = r.SetQueryParam("api_key", "k")

		return nil
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"X-Api-Key", "X-Changed"}, r.credentials.Headers)
	assert.Equal(t, []string{"api_key"}, r.credentials.Queries)

	names, ok := CredentialNamesFromContext(WithCredentialNames(context.Background(), r.credentials))
	require.True(t, ok)
	assert.Equal(t, r.credentials, names)

	_, ok = CredentialNamesFromContext(WithCredentialNames(context.Background(), CredentialNames{}))
	assert.False(t, ok)
}

func TestRecordCredentialsError(t *testing.T) {
	r := New("GET", "/x", nil)
	errAuth := errors.New("auth")

	err := r.recordCredentials(func() error { return errAuth })
	require.ErrorIs(t, err, errAuth)
	assert.Empty(t, r.credentials.Headers)
}
