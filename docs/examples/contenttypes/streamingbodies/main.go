// SPDX-License-Identifier: Apache-2.0

// Command streamingbodies backs the snippets on the doc-site
// "Streaming bodies" example page. Each function below is the source of a
// `{{< code region="..." >}}` include; the package as a whole compiles
// and lints so the snippets cannot rot silently.
//
// `go run .` is a no-op — every demo here either configures stateful
// API objects or expects a live HTTP server. Pass `serve` to actually
// start the untyped HTTP server demo (which blocks on ListenAndServe).
package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/client"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/runtime/middleware/untyped"
	"github.com/go-openapi/strfmt"
)

// --- Stubs (excluded from rendered snippets) ------------------------

// api and rt are package-level stand-ins so the snippet bodies read
// like in-context code. They stay nil — every demo function bails
// early when its prerequisites are missing, so `go run .` stays cheap.
var (
	api *untyped.API
	rt  *client.Runtime
	ctx = context.Background()
)

// putBackupParams is the bound parameter struct the untyped runtime
// would synthesize from the spec. Only the `Blob` field is exercised
// by the upload snippet.
type putBackupParams struct {
	Blob io.ReadCloser
}

// putBackupRequest writes the streaming body for the client snippet.
type putBackupRequest struct {
	body io.Reader
}

func (p putBackupRequest) WriteToRequest(req runtime.ClientRequest, _ strfmt.Registry) error {
	return req.SetBodyParam(p.body)
}

// putBackupResponse reads the (empty) response body for the client snippet.
type putBackupResponse struct{}

func (putBackupResponse) ReadResponse(_ runtime.ClientResponse, _ runtime.Consumer) (any, error) {
	return struct{}{}, nil
}

// use silences "declared and not used" diagnostics for snippet-local
// values that exist only to make the demo compile.
func use(_ ...any) {}

const readHeaderTimeout = 5 * time.Second

func main() {
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		serverDownload()
	}
	consumerWithCloses()
	serverUpload()
	clientStream()
}

// --- Snippets -------------------------------------------------------

func serverDownload() {
	doc, _ := loads.Spec("api.yaml")
	// snippet:serverDownload
	api := untyped.NewAPI(doc).WithJSONDefaults()

	// ByteStreamProducer is registered by WithJSONDefaults under
	// runtime.DefaultMime ("application/octet-stream"), but be explicit
	// when more than one stream-producing MIME is in the picture:
	api.RegisterProducer(runtime.DefaultMime, runtime.ByteStreamProducer())

	api.RegisterOperation("get", "/backups/{id}", runtime.OperationHandlerFunc(
		func(_ any) (any, error) {
			f, err := os.Open("/var/backups/2026-05-10.tar")
			if err != nil {
				return nil, err
			}
			// The Producer copies whatever io.Reader you return into the
			// response writer. Returning *os.File is fine; close it from
			// a Responder if you need ownership semantics.
			return middleware.ResponderFunc(func(w http.ResponseWriter, p runtime.Producer) {
				defer f.Close()
				w.Header().Set("Content-Type", runtime.DefaultMime)
				_ = p.Produce(w, f)
			}), nil
		},
	))
	// endsnippet:serverDownload

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           middleware.Serve(doc, api),
		ReadHeaderTimeout: readHeaderTimeout,
	}
	log.Fatal(srv.ListenAndServe())
}

func consumerWithCloses() {
	if api == nil {
		return
	}
	// snippet:consumerWithCloses
	api.RegisterConsumer(runtime.DefaultMime, runtime.ByteStreamConsumer(
		runtime.ClosesStream, // closes the io.ReadCloser when done
	))
	// endsnippet:consumerWithCloses
}

func serverUpload() {
	if api == nil {
		return
	}
	// snippet:serverUpload
	api.RegisterOperation("post", "/backups", runtime.OperationHandlerFunc(
		func(params any) (any, error) {
			body := params.(putBackupParams).Blob // io.ReadCloser
			defer body.Close()

			f, err := os.CreateTemp("", "upload-*")
			if err != nil {
				return nil, err
			}
			defer f.Close()

			if _, err := io.Copy(f, body); err != nil {
				return nil, err
			}
			return map[string]string{"status": "ok"}, nil
		},
	))
	// endsnippet:serverUpload
}

func clientStream() {
	if rt == nil {
		// keep the demo non-blocking: provide a fake reader instead of
		// touching the filesystem, and bail before the real Submit call.
		body := strings.NewReader("fake backup contents")
		op := &runtime.ClientOperation{
			ID:                 "PutBackup",
			Method:             "POST",
			PathPattern:        "/backups",
			ConsumesMediaTypes: []string{runtime.DefaultMime},
			Params:             putBackupRequest{body: body},
			Reader:             putBackupResponse{},
		}
		use(op)
		return
	}
	// snippet:clientStream
	file, _ := os.Open("./backup.tar")
	defer file.Close()

	op := &runtime.ClientOperation{
		ID:                 "PutBackup",
		Method:             "POST",
		PathPattern:        "/backups",
		ConsumesMediaTypes: []string{runtime.DefaultMime},
		Params:             putBackupRequest{body: file},
		Reader:             putBackupResponse{},
	}
	_, err := rt.SubmitContext(ctx, op)
	// endsnippet:clientStream

	use(err)
}
