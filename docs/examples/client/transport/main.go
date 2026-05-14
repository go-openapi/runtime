// SPDX-License-Identifier: Apache-2.0

// Command transport backs the snippets on the doc-site
// "Transport" page. Each function below is the source of a
// `{{< code region="..." >}}` include; the package as a whole compiles
// and lints so the snippets cannot rot silently.
//
// `go run .` exercises the non-blocking demos (construction and wiring).
// No outbound HTTP calls are issued.
package main

import (
	"log"
	"net/http"
	"net/url"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/client"
)

// --- Stubs (excluded from rendered snippets) ------------------------

// host, base, schemes stand in for the values a generated client would
// pass to client.NewWithClient. They are referenced from snippets that
// would otherwise need full literals on every line.
var (
	host    = "api.example.com"
	base    = "/v1"
	schemes = []string{"https"} //nolint:goconst // doc example: kept literal for snippet readability
)

// use silences "declared and not used" diagnostics for snippet-local
// values that exist only to make the demo compile.
func use(_ ...any) {}

func main() {
	registerVendorCodec()
	if _, err := setupMutualTLS(); err != nil {
		log.Printf("TLS setup (expected to fail without certs): %v", err)
	}
	timeoutClient()
	proxyFromEnv()
	proxyExplicit()
	enableConnectionReuse()
}

// --- Snippets -------------------------------------------------------

func registerVendorCodec() {
	// snippet:registerVendorCodec
	rt := client.New("api.example.com", "/v1", []string{"https"})
	rt.Consumers["application/vnd.acme.v1+json"] = runtime.JSONConsumer()
	rt.Producers["application/vnd.acme.v1+json"] = runtime.JSONProducer()
	// endsnippet:registerVendorCodec

	use(rt)
}

func setupMutualTLS() (*client.Runtime, error) {
	// snippet:setupMutualTLS
	tlsCfg, err := client.TLSClientAuth(client.TLSClientOptions{
		Certificate: "/etc/ssl/client.pem",
		Key:         "/etc/ssl/client-key.pem",
		CA:          "/etc/ssl/ca.pem",
		ServerName:  "api.internal",
	})
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Transport: &http.Transport{TLSClientConfig: tlsCfg},
	}
	rt := client.NewWithClient("api.internal", "/v1", []string{"https"}, httpClient)
	// endsnippet:setupMutualTLS

	return rt, nil
}

func timeoutClient() {
	// snippet:timeoutClient
	httpClient := &http.Client{Timeout: client.DefaultTimeout}
	rt := client.NewWithClient(host, base, schemes, httpClient)
	// endsnippet:timeoutClient

	use(rt)
}

func proxyFromEnv() {
	// snippet:proxyFromEnv
	// Honour HTTPS_PROXY / HTTP_PROXY (default behaviour anyway).
	tr := &http.Transport{Proxy: http.ProxyFromEnvironment}

	httpClient := &http.Client{Transport: tr}
	rt := client.NewWithClient(host, base, schemes, httpClient)
	// endsnippet:proxyFromEnv

	use(rt)
}

func proxyExplicit() {
	// snippet:proxyExplicit
	// Force a specific proxy.
	proxyURL, _ := url.Parse("http://proxy.internal:3128")
	tr := &http.Transport{Proxy: http.ProxyURL(proxyURL)}

	httpClient := &http.Client{Transport: tr}
	rt := client.NewWithClient(host, base, schemes, httpClient)
	// endsnippet:proxyExplicit

	use(rt)
}

func enableConnectionReuse() {
	// snippet:enableConnectionReuse
	rt := client.New("api.example.com", "/v1", []string{"https"})
	rt.EnableConnectionReuse()
	// endsnippet:enableConnectionReuse

	use(rt)
}
