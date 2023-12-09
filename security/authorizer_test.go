// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package security

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthorized(t *testing.T) {
	authorizer := Authorized()

	err := authorizer.Authorize(nil, nil)
	require.NoError(t, err)
}

func TestAuthenticator(t *testing.T) {
	r, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	require.NoError(t, err)

	t.Run("with HttpAuthenticator", func(t *testing.T) {
		auth := HttpAuthenticator(func(_ *http.Request) (bool, interface{}, error) { return true, "test", nil })

		t.Run("authenticator should work on *http.Request", func(t *testing.T) {
			isAuth, user, err := auth.Authenticate(r)
			require.NoError(t, err)
			assert.True(t, isAuth)
			assert.Equal(t, "test", user)
		})

		t.Run("authenticator should work on *ScopedAuthRequest", func(t *testing.T) {
			scoped := &ScopedAuthRequest{Request: r}

			isAuth, user, err := auth.Authenticate(scoped)
			require.NoError(t, err)
			assert.True(t, isAuth)
			assert.Equal(t, "test", user)
		})

		t.Run("authenticator should return false on other inputs", func(t *testing.T) {
			isAuth, user, err := auth.Authenticate("")
			require.NoError(t, err)
			assert.False(t, isAuth)
			assert.Empty(t, user)
		})
	})

	t.Run("with ScopedAuthenticator", func(t *testing.T) {
		auth := ScopedAuthenticator(func(_ *ScopedAuthRequest) (bool, interface{}, error) { return true, "test", nil })

		t.Run("authenticator should work on *ScopedAuthRequest", func(t *testing.T) {
			scoped := &ScopedAuthRequest{Request: r}

			isAuth, user, err := auth.Authenticate(scoped)
			require.NoError(t, err)
			assert.True(t, isAuth)
			assert.Equal(t, "test", user)
		})

		t.Run("authenticator should return false on other inputs", func(t *testing.T) {
			isAuth, user, err := auth.Authenticate("")
			require.NoError(t, err)
			assert.False(t, isAuth)
			assert.Empty(t, user)
		})
	})
}
