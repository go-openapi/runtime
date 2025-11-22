package middleware

import (
	stdcontext "context"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-openapi/runtime/internal/testing/petstore"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestRouterContext_Issue375(t *testing.T) {
	spec, api := petstore.NewAPI(t)
	spec.Spec().BasePath = "/api/"
	context := NewContext(spec, api, nil)

	type authCtxKey uint8
	const authCtx authCtxKey = iota + 1
	authCtxErr := stderrors.New("test error in context")

	mw := NewRouter(context, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// check context after API middleware
		authContext := stdcontext.WithValue(r.Context(), authCtx, authCtxErr)
		*r = *r.WithContext(authContext)
	}))

	start := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("calling API router with context: %v", authCtxErr)
		mw.ServeHTTP(w, r)

		value := r.Context().Value(authCtx)
		assert.NotNilf(t, value, "end of middleware chain: expected to find authCtx in request context")

		if value == nil {
			w.WriteHeader(http.StatusInternalServerError)
		}

		errAuth, ok := value.(error)
		assert.Truef(t, ok, "expected authCtx to be an error, but got: %T", value)
		t.Logf("got context from request: %v", errAuth)
		fmt.Fprintf(w, "%v", errAuth)
		w.WriteHeader(http.StatusOK)

		// *r = *r.WithContext(authContext)
		// mw.ServeHTTP(w, r)
	})

	recorder := httptest.NewRecorder()
	request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "/api/pets/123", nil)
	require.NoError(t, err)

	start.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusOK, recorder.Code)

	res := recorder.Result()
	require.NotNil(t, res.Body)
	msg, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	t.Logf("response message: %q", string(msg))
}
