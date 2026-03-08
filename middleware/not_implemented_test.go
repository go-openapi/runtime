// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/testify/v2/require"
)

func TestErrorResponder(t *testing.T) {
	resp := Error(http.StatusBadRequest, map[string]string{"message": "this is the error body"})

	rec := httptest.NewRecorder()
	resp.WriteResponse(rec, runtime.JSONProducer())

	require.EqualT(t, http.StatusBadRequest, rec.Code)
	require.JSONEqT(t, "{\"message\":\"this is the error body\"}\n", rec.Body.String())
}
