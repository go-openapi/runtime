// SPDX-License-Identifier: Apache-2.0

// Command auth backs the snippets on the doc-site
// "Client / Authentication" page. Each function below is the source of a
// `{{< code region="..." >}}` include; the package as a whole compiles
// and lints so the snippets cannot rot silently.
//
// `go run .` exercises the non-blocking demos (wiring registration).
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

// --- Stubs (excluded from rendered snippets) ------------------------

// rt is a placeholder client runtime. Snippets pretend it was built
// via client.New(...); the demo wires a fresh empty runtime so the
// program compiles and runs.
var rt = client.New("example.com", "/", []string{"https"})

// op is a placeholder per-operation client request descriptor.
var op = &runtime.ClientOperation{}

// token, accessToken and apiKey stand in for opaque credentials read
// from configuration or a secret manager.
const (
	token       = "demo-token"
	accessToken = "demo-access-token"
	apiKey      = "demo-api-key" //nolint:gosec // doc example fixture
)

// use silences "declared and not used" diagnostics for snippet-local
// values that exist only to make the demo compile.
func use(_ ...any) {}

func main() {
	attachAuth()
	basicAuth()
	apiKeyAuth()
	bearerAuth()
	composeAuth()
	passThroughAuth()
	useHMAC()
}

// --- Snippets -------------------------------------------------------

func attachAuth() {
	// snippet:attachAuth
	// 1. Per operation — overrides the runtime default
	op.AuthInfo = client.BearerToken(token)

	// 2. Per runtime — used when the operation does not set its own
	rt.DefaultAuthentication = client.BasicAuth("alice", "s3cret")
	// endsnippet:attachAuth
}

func basicAuth() {
	// snippet:basicAuth
	rt.DefaultAuthentication = client.BasicAuth("alice", "s3cret")
	// endsnippet:basicAuth
}

func apiKeyAuth() {
	// snippet:apiKeyAuth
	// As an HTTP header
	rt.DefaultAuthentication = client.APIKeyAuth("X-Api-Key", "header", apiKey)

	// Or as a query parameter
	rt.DefaultAuthentication = client.APIKeyAuth("api_key", "query", apiKey)
	// endsnippet:apiKeyAuth
}

func bearerAuth() {
	// snippet:bearerAuth
	rt.DefaultAuthentication = client.BearerToken(accessToken)
	// endsnippet:bearerAuth
}

func composeAuth() {
	// snippet:composeAuth
	rt.DefaultAuthentication = client.Compose(
		client.APIKeyAuth("X-Api-Key", "header", apiKey),
		client.BearerToken(accessToken),
	)
	// endsnippet:composeAuth
}

func passThroughAuth() {
	// snippet:passThroughAuth
	op.AuthInfo = client.PassThroughAuth
	// endsnippet:passThroughAuth
}

func useHMAC() {
	rt.DefaultAuthentication = HMACSignature("key-1", []byte("shared-secret"))
	use(rt)
}

// snippet:hmacSignature

// HMACSignature returns a ClientAuthInfoWriter that signs the request
// body with the given HMAC-SHA256 key and attaches the signature plus
// key ID as headers.
func HMACSignature(keyID string, key []byte) runtime.ClientAuthInfoWriter {
	return runtime.ClientAuthInfoWriterFunc(func(r runtime.ClientRequest, _ strfmt.Registry) error {
		body := r.GetBody()
		mac := hmac.New(sha256.New, key)
		mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))
		if err := r.SetHeaderParam("X-Sig-Key", keyID); err != nil {
			return err
		}
		return r.SetHeaderParam("X-Sig", sig)
	})
}

// endsnippet:hmacSignature
