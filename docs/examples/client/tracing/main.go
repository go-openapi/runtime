// SPDX-License-Identifier: Apache-2.0

// Command tracing backs the snippets on the doc-site
// "Client / Tracing" page. Each function below is the source of a
// `{{< code region="..." >}}` include; the package as a whole compiles
// and lints so the snippets cannot rot silently.
//
// `go run .` exercises the non-blocking demos (wiring registration —
// no real HTTP traffic is made).
package main

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

const samplePetID = 42

// --- Stubs (excluded from rendered snippets) ------------------------

// petclient is a stand-in for a go-swagger-generated client package
// (e.g. example.com/petstore/client). It is declared in this file so the
// snippet renders as written without inventing an import path that does
// not exist in the example module.
var petclient = fakePetClientPkg{}

type fakePetClientPkg struct{}

// New mimics the generated petclient.New(transport, formats) entry point.
func (fakePetClientPkg) New(_ runtime.ClientTransport, _ strfmt.Registry) *fakePetClient {
	return &fakePetClient{}
}

// NewGetPetParams mimics the generated params constructor.
func (fakePetClientPkg) NewGetPetParams() *fakeGetPetParams { return &fakeGetPetParams{} }

type fakePetClient struct {
	Operations fakeOperations
}

type fakeOperations struct{}

func (fakeOperations) GetPet(_ *fakeGetPetParams) (any, error) { return struct{}{}, nil }

type fakeGetPetParams struct{ ID int64 }

func (p *fakeGetPetParams) WithID(id int64) *fakeGetPetParams { p.ID = id; return p }

// use silences "declared and not used" diagnostics for snippet-local
// values that exist only to make the demo compile.
func use(_ ...any) {}

func main() {
	wireOpenTelemetry()
	customSpanFormatter()
}

// --- Snippets -------------------------------------------------------

func wireOpenTelemetry() {
	// snippet:wireOpenTelemetry
	rt := client.New("api.example.com", "/v1", []string{"https"})
	traced := rt.WithOpenTelemetry()

	api := petclient.New(traced, strfmt.Default)
	result, err := api.Operations.GetPet(petclient.NewGetPetParams().WithID(samplePetID))
	// endsnippet:wireOpenTelemetry

	use(traced, api, result, err)
}

func customSpanFormatter() {
	rt := client.New("api.example.com", "/v1", []string{"https"})

	// snippet:customSpanFormatter
	traced := rt.WithOpenTelemetry(
		client.WithSpanNameFormatter(func(op *runtime.ClientOperation) string {
			return "petstore." + op.ID
		}),
		client.WithSpanOptions(
			trace.WithAttributes(
				attribute.String("service.name", "petstore-client"),
			),
		),
	)
	// endsnippet:customSpanFormatter

	use(traced)
}
