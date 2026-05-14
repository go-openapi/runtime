// SPDX-License-Identifier: Apache-2.0

// Command contenttyper backs the snippets on the doc-site
// "Per-payload Content-Type override" page. Each function below is the
// source of a `{{< code region="..." >}}` include; the package as a
// whole compiles and lints so the snippets cannot rot silently.
//
// `go run .` exercises the non-blocking demos (no network is touched —
// the fake transport just records the picked Content-Type).
package main

import (
	"io"
	"os"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// --- Stubs (excluded from rendered snippets) ------------------------

// fakeTransport stands in for an application's runtime.ClientTransport
// (typically a client.Runtime). Submit is a no-op so the demos can run
// without a real server.
type fakeTransport struct{}

func (fakeTransport) Submit(_ *runtime.ClientOperation) (any, error) {
	return struct{}{}, nil
}

// putAvatarBody stands in for a generated request-writer builder. In
// generated client code this would be produced by go-swagger from the
// operation's body parameter and would attach the payload via
// ClientRequest.SetBodyParam.
func putAvatarBody(body any) runtime.ClientRequestWriter {
	return runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		return req.SetBodyParam(body)
	})
}

// putAvatarReader stands in for a generated response reader.
type putAvatarReader struct{}

func (putAvatarReader) ReadResponse(_ runtime.ClientResponse, _ runtime.Consumer) (any, error) {
	return struct{}{}, nil
}

// use silences "declared and not used" diagnostics for snippet-local
// values that exist only to make the demo compile.
func use(_ ...any) {}

func main() {
	streamPayloadDemo()
	multipartFilePartDemo()
}

func streamPayloadDemo() {
	// Use a throwaway file so the demo runs without an external avatar.
	tmp, err := os.CreateTemp("", "avatar-*.png")
	if err != nil {
		return
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	_ = uploadAvatar(fakeTransport{}, tmp.Name())
}

func multipartFilePartDemo() {
	tmp, err := os.CreateTemp("", "manifest-*.json")
	if err != nil {
		return
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	f, _ := os.Open(tmp.Name())
	defer f.Close()
	part := taggedFile{File: f, mime: "application/vnd.acme.manifest+json"}
	use(part)
}

// --- Snippets -------------------------------------------------------

// snippet:streamPayload
type imagePayload struct {
	body io.Reader
	mime string
}

func (p imagePayload) Read(b []byte) (int, error) { return p.body.Read(b) }

// ContentTyper — wins over the operation's `consumes` default.
func (p imagePayload) ContentType() string { return p.mime }

func uploadAvatar(rt runtime.ClientTransport, avatar string) error {
	f, _ := os.Open(avatar)
	defer f.Close()

	op := &runtime.ClientOperation{
		ID:          "UploadAvatar",
		Method:      "PUT",
		PathPattern: "/users/me/avatar",
		Params: putAvatarBody(imagePayload{
			body: f,
			mime: "image/png", // ← will land on the wire as Content-Type
		}),
		Reader: putAvatarReader{},
	}
	_, err := rt.Submit(op)
	return err
}

// endsnippet:streamPayload

// snippet:multipartFileType
type taggedFile struct {
	*os.File

	mime string
}

func (t taggedFile) ContentType() string { return t.mime }

// endsnippet:multipartFileType
