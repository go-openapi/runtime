// SPDX-License-Identifier: Apache-2.0

// Command customcodec backs the snippets on the doc-site
// "Custom codec (MessagePack)" recipe page. Each region below is the
// source of a `{{< code region="..." >}}` include; the package as a
// whole compiles and lints so the snippets cannot rot silently.
//
// The example uses MessagePack as the worked content-type, but the
// real `github.com/vmihailenco/msgpack/v5` dependency is replaced by
// a tiny in-file shim so this recipe doesn't add a third-party module
// to the examples go.mod. The codec contract (Consumer / Producer) is
// what the recipe is about — the wire format is incidental.
//
// `go run .` exercises the non-blocking demos (registration wiring).
package main

import (
	"io"

	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/client"
	"github.com/go-openapi/runtime/middleware/untyped"
)

// --- Stubs (excluded from rendered snippets) ------------------------

// doc is a placeholder OpenAPI document. The snippets pretend it was
// loaded from disk; the demo wires nil so the program compiles and
// runs without an on-disk spec.
var doc *loads.Document

// op stands in for a generated client operation whose media-type lists
// the caller may want to override per-call.
var op = &runtime.ClientOperation{}

// msgpack mimics the surface of github.com/vmihailenco/msgpack/v5 used
// by the recipe (NewDecoder / NewEncoder). It exists only so the
// snippet bodies compile without pulling the real dependency into the
// examples module. Real applications import vmihailenco/msgpack/v5.
var msgpack = msgpackShim{}

type msgpackShim struct{}

func (msgpackShim) NewDecoder(r io.Reader) *msgpackDecoder { return &msgpackDecoder{r: r} }
func (msgpackShim) NewEncoder(w io.Writer) *msgpackEncoder { return &msgpackEncoder{w: w} }

type msgpackDecoder struct{ r io.Reader }

func (d *msgpackDecoder) Decode(_ any) error {
	// Drain the reader so the stub behaves like a real decoder w.r.t. io.
	_, err := io.Copy(io.Discard, d.r)
	return err
}

type msgpackEncoder struct{ w io.Writer }

func (e *msgpackEncoder) Encode(_ any) error {
	_, err := e.w.Write(nil)
	return err
}

// use silences "declared and not used" diagnostics for snippet-local
// values that exist only to make the demo compile.
func use(_ ...any) {}

func main() {
	registerOnServer()
	registerOnClient()
	operationMediaTypes()
}

// --- Snippets -------------------------------------------------------

// snippet:consumerProducerPair

// Mime is the content-type the recipe registers under. MessagePack has
// no IANA-registered MIME; application/x-msgpack and application/msgpack
// are both common — pick one and stick to it.
const Mime = "application/x-msgpack"

// Consumer returns a runtime.Consumer that decodes a MessagePack body
// into the target value v.
func Consumer() runtime.Consumer {
	return runtime.ConsumerFunc(func(r io.Reader, v any) error {
		return msgpack.NewDecoder(r).Decode(v)
	})
}

// Producer returns a runtime.Producer that serialises v as MessagePack
// onto w.
func Producer() runtime.Producer {
	return runtime.ProducerFunc(func(w io.Writer, v any) error {
		return msgpack.NewEncoder(w).Encode(v)
	})
}

// endsnippet:consumerProducerPair

func registerOnServer() {
	// snippet:registerOnServer
	api := untyped.NewAPI(doc).WithJSONDefaults() // JSON codecs registered for free
	api.RegisterConsumer(Mime, Consumer())
	api.RegisterProducer(Mime, Producer())
	// endsnippet:registerOnServer

	use(api)
}

func registerOnClient() {
	// snippet:registerOnClient
	rt := client.New("api.example.com", "/v1", []string{"https"})
	rt.Consumers[Mime] = Consumer()
	rt.Producers[Mime] = Producer()
	// endsnippet:registerOnClient

	use(rt)
}

func operationMediaTypes() {
	// snippet:operationMediaTypes
	op.ConsumesMediaTypes = []string{Mime}
	op.ProducesMediaTypes = []string{Mime}
	// endsnippet:operationMediaTypes
}
