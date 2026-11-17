// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	"github.com/go-openapi/runtime"
)

const (
	testHeaderKey = "HEADER_KEY_MUST_NOT_LEAK"
	testQueryKey  = "QUERY_KEY_MUST_NOT_LEAK"
	testBearer    = "BEARER_MUST_NOT_LEAK"

	testOrigin      = "api.test/thing" // route of the operation on the configured host
	testTrustedHost = "api-eu.test"
	testAPIURL      = "http://api.test"
)

// redirectLab serves any host name, over http on every port but 443 and over https on 443.
//
// routes maps "host/path" to the Location of a 302. In a Location, "{query}" stands for the
// query of the request. A request without a route lands: it is recorded and answered with 200.
type redirectLab struct {
	plain, secure *httptest.Server
	routes        map[string]string

	mu     sync.Mutex
	landed []*http.Request
}

func newRedirectLab(t *testing.T, routes map[string]string) *redirectLab {
	t.Helper()

	lab := &redirectLab{routes: routes}
	lab.plain = httptest.NewServer(lab)
	lab.secure = httptest.NewTLSServer(lab)
	t.Cleanup(lab.plain.Close)
	t.Cleanup(lab.secure.Close)

	return lab
}

func (lab *redirectLab) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}

	if location, ok := lab.routes[host+r.URL.Path]; ok {
		http.Redirect(w, r, strings.ReplaceAll(location, "{query}", r.URL.RawQuery), http.StatusFound) //nolint:gosec // the test server redirects on purpose

		return
	}

	lab.mu.Lock()
	lab.landed = append(lab.landed, r.Clone(r.Context()))
	lab.mu.Unlock()

	w.Header().Set(runtime.HeaderContentType, runtime.JSONMime)
	_, _ = w.Write([]byte(`{}`))
}

func (lab *redirectLab) transport() http.RoundTripper {
	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			_, port, _ := net.SplitHostPort(addr)
			target := lab.plain.Listener.Addr().String()
			if port == "443" {
				target = lab.secure.Listener.Addr().String()
			}

			return (&net.Dialer{}).DialContext(ctx, network, target)
		},
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // httptest servers use a self-signed certificate
	}
}

func (lab *redirectLab) runtime(scheme string) *Runtime {
	rt := New("api.test", "/", []string{scheme})
	rt.Transport = lab.transport()

	return rt
}

func (lab *redirectLab) lastLanded(t *testing.T) *http.Request {
	t.Helper()

	lab.mu.Lock()
	defer lab.mu.Unlock()
	require.NotEmpty(t, lab.landed, "the redirect must be followed")

	return lab.landed[len(lab.landed)-1]
}

func redirectOperation(scheme string) *runtime.ClientOperation {
	return &runtime.ClientOperation{
		ID:                 "redirectTest",
		Method:             http.MethodGet,
		PathPattern:        "/thing",
		ProducesMediaTypes: []string{runtime.JSONMime},
		ConsumesMediaTypes: []string{runtime.JSONMime},
		Schemes:            []string{scheme},
		Params: runtime.ClientRequestWriterFunc(func(runtime.ClientRequest, strfmt.Registry) error {
			return nil
		}),
		Reader: runtime.ClientResponseReaderFunc(func(runtime.ClientResponse, runtime.Consumer) (any, error) {
			return nil, nil
		}),
		AuthInfo: Compose(
			APIKeyAuth("X-Api-Key", "header", testHeaderKey),
			APIKeyAuth("api_key", "query", testQueryKey),
			BearerToken(testBearer),
		),
	}
}

func TestRedirectCredentials(t *testing.T) {
	for _, tc := range []struct {
		name     string
		scheme   string
		routes   map[string]string
		trusted  []string
		wantHost string
		wantKept bool
	}{
		{
			name:     "same host",
			scheme:   "http",
			routes:   map[string]string{testOrigin: testAPIURL + "/landed?{query}"},
			wantHost: "api.test",
			wantKept: true,
		},
		{
			name:     "same host, other port",
			scheme:   "http",
			routes:   map[string]string{testOrigin: "http://api.test:8080/landed?{query}"},
			wantHost: "api.test:8080",
			wantKept: true,
		},
		{
			name:     "subdomain",
			scheme:   "http",
			routes:   map[string]string{testOrigin: "http://eu.api.test/landed?{query}"},
			wantHost: "eu.api.test",
			wantKept: true,
		},
		{
			name:     "upgrade from http to https",
			scheme:   "http",
			routes:   map[string]string{testOrigin: "https://api.test/landed?{query}"},
			wantHost: "api.test",
			wantKept: true,
		},
		{
			name:     "other host",
			scheme:   "http",
			routes:   map[string]string{testOrigin: "http://evil.test/landed?{query}"},
			wantHost: "evil.test",
		},
		{
			name:     "other host, Location without the query",
			scheme:   "http",
			routes:   map[string]string{testOrigin: "http://evil.test/landed"},
			wantHost: "evil.test",
		},
		{
			name:     "parent domain",
			scheme:   "http",
			routes:   map[string]string{testOrigin: "http://test/landed?{query}"},
			wantHost: "test",
		},
		{
			name:     "downgrade from https to http on the same host",
			scheme:   "https",
			routes:   map[string]string{testOrigin: testAPIURL + "/landed?{query}"},
			wantHost: "api.test",
		},
		{
			name:     "trusted host",
			scheme:   "http",
			routes:   map[string]string{testOrigin: "http://api-eu.test/landed?{query}"},
			trusted:  []string{testTrustedHost},
			wantHost: "api-eu.test",
			wantKept: true,
		},
		{
			name:     "subdomain of a trusted host",
			scheme:   "http",
			routes:   map[string]string{testOrigin: "http://v2.api-eu.test/landed?{query}"},
			trusted:  []string{"API-EU.test"},
			wantHost: "v2.api-eu.test",
			wantKept: true,
		},
		{
			name:     "downgrade from https to http on a trusted host",
			scheme:   "https",
			routes:   map[string]string{testOrigin: "http://api-eu.test/landed?{query}"},
			trusted:  []string{testTrustedHost},
			wantHost: "api-eu.test",
		},
		{
			name:   "chain that leaves the host and comes back",
			scheme: "http",
			routes: map[string]string{
				testOrigin:         "http://evil.test/bounce?{query}",
				"evil.test/bounce": testAPIURL + "/landed?{query}",
			},
			wantHost: "api.test",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lab := newRedirectLab(t, tc.routes)
			rt := lab.runtime(tc.scheme)
			rt.TrustedRedirectHosts = tc.trusted

			_, err := rt.Submit(redirectOperation(tc.scheme))
			require.NoError(t, err)

			got := lab.lastLanded(t)
			assert.Equal(t, tc.wantHost, got.Host)
			assert.Equal(t, "/landed", got.URL.Path)
			assert.NotEmpty(t, got.Header.Get(runtime.HeaderAccept), "other headers follow the redirect")

			if tc.wantKept {
				assert.Equal(t, testHeaderKey, got.Header.Get("X-Api-Key"))
				assert.Equal(t, "Bearer "+testBearer, got.Header.Get(runtime.HeaderAuthorization))
				assert.Equal(t, testQueryKey, got.URL.Query().Get("api_key"))

				return
			}

			assert.Empty(t, got.Header.Get("X-Api-Key"))
			assert.Empty(t, got.Header.Get(runtime.HeaderAuthorization))
			assert.NotContains(t, got.URL.RawQuery, testQueryKey)
			assert.Empty(t, got.Header.Get("Referer"), "Referer carries the previous URL, query included")
		})
	}
}

func TestRedirectKeepsOtherQueryParams(t *testing.T) {
	lab := newRedirectLab(t, map[string]string{
		// url.Values.Encode would re-order "z", re-escape "%2F" and drop "bad=%zz"
		testOrigin: "http://evil.test/landed?z=%2F&{query}&bad=%zz&flag",
	})

	_, err := lab.runtime("http").Submit(redirectOperation("http"))
	require.NoError(t, err)

	assert.Equal(t, "z=%2F&bad=%zz&flag", lab.lastLanded(t).URL.RawQuery)
}

func TestRedirectLimit(t *testing.T) {
	lab := newRedirectLab(t, map[string]string{testOrigin: "http://api.test/thing"})

	_, err := lab.runtime("http").Submit(redirectOperation("http"))
	require.Error(t, err)
	require.ErrorIs(t, err, errTooManyRedirects)
}

func TestRedirectSuppliedClient(t *testing.T) {
	routes := map[string]string{testOrigin: "http://evil.test/landed?{query}"}

	t.Run("keeps its own CheckRedirect", func(t *testing.T) {
		lab := newRedirectLab(t, routes)
		client := &http.Client{
			Transport: lab.transport(),
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		rt := NewWithClient("api.test", "/", []string{"http"}, client)

		_, _ = rt.Submit(redirectOperation("http"))

		assert.Empty(t, lab.landed)
	})

	t.Run("without CheckRedirect gets the default policy of net/http", func(t *testing.T) {
		lab := newRedirectLab(t, routes)
		rt := NewWithClient("api.test", "/", []string{"http"}, &http.Client{Transport: lab.transport()})

		_, err := rt.Submit(redirectOperation("http"))
		require.NoError(t, err)

		got := lab.lastLanded(t)
		assert.Empty(t, got.Header.Get(runtime.HeaderAuthorization))
		assert.Equal(t, testHeaderKey, got.Header.Get("X-Api-Key"))
	})
}

func TestDeleteQueryParams(t *testing.T) {
	for _, tc := range []struct {
		raw   string
		names []string
		want  string
	}{
		{raw: "", names: []string{"k"}, want: ""},
		{raw: "a=1&k=2", names: nil, want: "a=1&k=2"},
		{raw: "a=1&k=2&b=3", names: []string{"k"}, want: "a=1&b=3"},
		{raw: "k=1&k=2", names: []string{"k"}, want: ""},
		{raw: "a%20b=1&c=2", names: []string{"a b"}, want: "c=2"},
		{raw: "K=1&k=2", names: []string{"k"}, want: "K=1"},
		{raw: "k&a=%zz", names: []string{"k"}, want: "a=%zz"},
	} {
		u := &url.URL{RawQuery: tc.raw}
		deleteQueryParams(u, tc.names)
		assert.Equal(t, tc.want, u.RawQuery, "query %q without %v", tc.raw, tc.names)
	}
}

func TestCredentialsMayFollow(t *testing.T) {
	rt := New("api.test", "/", nil)
	rt.TrustedRedirectHosts = []string{"", "trusted.test"}
	parse := func(s string) *url.URL {
		u, err := url.Parse(s)
		require.NoError(t, err)

		return u
	}

	for _, tc := range []struct {
		initial, dest string
		want          bool
	}{
		{"http://API.test", "http://api.TEST", true},
		{testAPIURL, "http://xapi.test", false},
		{testAPIURL, "http://evil.test.", false}, // the empty trusted entry matches nothing
		{testAPIURL, "http://a.trusted.test", true},
		{"http://[::1]:8080", "http://[::1]:9090", true},
		{"http://example.com", "http://[::1%25.example.com]", false},
		{"https://api.test", "HTTP://api.test", false},
		{"HTTPS://api.test", "https://api.test", true},
	} {
		assert.Equal(t, tc.want, rt.credentialsMayFollow(parse(tc.initial), parse(tc.dest)), "%s -> %s", tc.initial, tc.dest)
	}
}
