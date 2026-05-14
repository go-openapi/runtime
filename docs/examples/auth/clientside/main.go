// SPDX-License-Identifier: Apache-2.0

// Command clientside backs the snippets on the doc-site
// "Client-side credentials" recipe page. Each function below is the
// source of a `{{< code region="..." >}}` include; the package as a
// whole compiles and lints so the snippets cannot rot silently.
//
// `go run .` exercises the non-blocking demos (writer construction and
// per-operation wiring).
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

// apiKey, accessToken, operationSpecificToken and sharedSecret stand in
// for real credentials. In a production client they would come from
// configuration or a secret store.
var (
	apiKey                 = "k-stub"
	accessToken            = "t-stub"
	operationSpecificToken = "op-stub"
	sharedSecret           = []byte("hmac-stub")
)

// op stands in for a *runtime.ClientOperation built by generated code.
// The snippets only touch its AuthInfo field.
var op = &runtime.ClientOperation{}

// use silences "declared and not used" diagnostics for snippet-local
// values that exist only to make the demo compile.
func use(_ ...any) {}

func main() {
	builtinWriters()
	perOperationOverride()
	composeWriters()
	hmacSignatureWriter()
	passThroughAuth()
}

// --- Snippets -------------------------------------------------------

func builtinWriters() {
	// snippet:builtinWriters
	rt := client.New("api.example.com", "/v1", []string{"https"}) //nolint:goconst // doc example

	// One of:
	rt.DefaultAuthentication = client.BasicAuth("alice", "s3cret")
	rt.DefaultAuthentication = client.APIKeyAuth("X-Api-Key", "header", apiKey)
	rt.DefaultAuthentication = client.APIKeyAuth("api_key", "query", apiKey)
	rt.DefaultAuthentication = client.BearerToken(accessToken)
	// endsnippet:builtinWriters

	use(rt)
}

func perOperationOverride() {
	// snippet:perOperationOverride
	op.AuthInfo = client.BearerToken(operationSpecificToken)
	// endsnippet:perOperationOverride
}

func composeWriters() {
	rt := client.New("api.example.com", "/v1", []string{"https"})

	// snippet:composeWriters
	rt.DefaultAuthentication = client.Compose(
		client.APIKeyAuth("X-Api-Key", "header", apiKey),
		client.BearerToken(accessToken),
	)
	// endsnippet:composeWriters

	use(rt)
}

// snippet:hmacSignatureWriter

// HMACSignature attaches an HMAC-SHA256 signature of the request body
// as `X-Sig`, along with the key identifier as `X-Sig-Key`.
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

// endsnippet:hmacSignatureWriter

func hmacSignatureWriter() {
	rt := client.New("api.example.com", "/v1", []string{"https"})
	rt.DefaultAuthentication = HMACSignature("k1", sharedSecret)

	use(rt)
}

func passThroughAuth() {
	// snippet:passThroughAuth
	op.AuthInfo = client.PassThroughAuth
	// endsnippet:passThroughAuth
}
