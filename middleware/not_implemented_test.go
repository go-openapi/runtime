package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-openapi/runtime"
	"github.com/stretchr/testify/require"
)

func TestErrorResponder(t *testing.T) {
	resp := Error(http.StatusBadRequest, map[string]string{"message": "this is the error body"})

	rec := httptest.NewRecorder()
	resp.WriteResponse(rec, runtime.JSONProducer())

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "{\"message\":\"this is the error body\"}\n", rec.Body.String())
}
