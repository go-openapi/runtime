package denco_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-openapi/runtime/middleware/denco"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testHandlerFunc(w http.ResponseWriter, r *http.Request, params denco.Params) {
	fmt.Fprintf(w, "method: %s, path: %s, params: %v", r.Method, r.URL.Path, params)
}

func TestMux(t *testing.T) {
	mux := denco.NewMux()
	handler, err := mux.Build([]denco.Handler{
		mux.GET("/", testHandlerFunc),
		mux.GET("/user/:name", testHandlerFunc),
		mux.POST("/user/:name", testHandlerFunc),
		mux.HEAD("/user/:name", testHandlerFunc),
		mux.PUT("/user/:name", testHandlerFunc),
		mux.Handler(http.MethodGet, "/user/handler", testHandlerFunc),
		mux.Handler(http.MethodPost, "/user/handler", testHandlerFunc),
		mux.Handler(http.MethodPut, "/user/inference", testHandlerFunc),
	})
	require.NoError(t, err)

	server := httptest.NewServer(handler)
	defer server.Close()

	for _, v := range []struct {
		status                 int
		method, path, expected string
	}{
		{http.StatusOK, http.MethodGet, "/", "method: GET, path: /, params: []"},
		{http.StatusOK, http.MethodGet, "/user/alice", "method: GET, path: /user/alice, params: [{name alice}]"},
		{http.StatusOK, http.MethodPost, "/user/bob", "method: POST, path: /user/bob, params: [{name bob}]"},
		{http.StatusOK, http.MethodHead, "/user/alice", ""},
		{http.StatusOK, http.MethodPut, "/user/bob", "method: PUT, path: /user/bob, params: [{name bob}]"},
		{http.StatusNotFound, http.MethodPost, "/", "404 page not found\n"},
		{http.StatusNotFound, http.MethodGet, "/unknown", "404 page not found\n"},
		{http.StatusNotFound, http.MethodPost, "/user/alice/1", "404 page not found\n"},
		{http.StatusOK, http.MethodGet, "/user/handler", "method: GET, path: /user/handler, params: []"},
		{http.StatusOK, http.MethodPost, "/user/handler", "method: POST, path: /user/handler, params: []"},
		{http.StatusOK, http.MethodPut, "/user/inference", "method: PUT, path: /user/inference, params: []"},
	} {
		req, err := http.NewRequestWithContext(context.Background(), v.method, server.URL+v.path, nil)
		require.NoError(t, err)

		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)

		actual := string(body)
		expected := v.expected

		assert.Equalf(t, v.status, res.StatusCode, "for method %s in path %s", v.method, v.path)
		assert.Equalf(t, expected, actual, "for method %s in path %s", v.method, v.path)
	}
}

func TestNotFound(t *testing.T) {
	mux := denco.NewMux()
	handler, err := mux.Build([]denco.Handler{})
	require.NoError(t, err)

	server := httptest.NewServer(handler)
	defer server.Close()

	origNotFound := denco.NotFound
	defer func() {
		denco.NotFound = origNotFound
	}()
	denco.NotFound = func(w http.ResponseWriter, r *http.Request, params denco.Params) {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, "method: %s, path: %s, params: %v", r.Method, r.URL.Path, params)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	actual := string(body)
	expected := "method: GET, path: /, params: []"

	assert.Equal(t, http.StatusServiceUnavailable, res.StatusCode)
	assert.Equal(t, expected, actual)
}
