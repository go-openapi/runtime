// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"iter"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// This file is a behavioural harness for the client-side content-type
// selection paths. It is intentionally exhaustive: each case captures
// what the runtime does today (correct behaviour and known bugs alike)
// so subsequent fixes for issues #386 and #387 produce visible deltas.
//
// Cases tagged with #386 lock in behaviour that is still known
// to be incorrect — they will be flipped when the picker becomes
// payload-aware.

const (
	ndjsonMime  = "application/x-ndjson"
	vendorMime  = "application/x-vendor"
	vendorMime1 = "application/x-vendor1"
	vendorMime2 = "application/x-vendor2"
)

// buildHTTPCase exercises (*request).buildHTTP directly: the picker
// has already chosen mediaType.
//
// Set wantContentType for an exact match. For multipart cases (where
// the boundary is random), set wantContentTypePrefix instead.
type buildHTTPCase struct {
	name                  string
	mediaType             string                          // already-picked mime
	consumes              []string                        // candidate list visible to buildHTTP (Stage-2 input)
	method                string                          // default POST when empty
	writer                runtime.ClientRequestWriter     // SetBodyParam / SetFormParam / SetFileParam
	producers             map[string]runtime.Producer     // nil → defaults from New()
	wantContentType       string                          // exact match; empty → no header expected unless prefix is set
	wantContentTypePrefix string                          // prefix match (use for multipart with random boundary)
	wantBody              func(t *testing.T, body []byte) // nil → no body assertion
	wantErr               string                          // substring of error
}

// submitCase exercises Runtime.Submit end-to-end via a captured
// RoundTripper: both the picker and buildHTTP run.
type submitCase struct {
	name                  string
	consumes              []string // operation.ConsumesMediaTypes
	producesAccept        []string // operation.ProducesMediaTypes (drives Accept header)
	method                string   // default POST when empty
	writer                runtime.ClientRequestWriter
	producers             map[string]runtime.Producer // nil → defaults
	wantContentType       string
	wantContentTypePrefix string
	wantBody              func(t *testing.T, body []byte)
	wantErr               string
}

func runBuildHTTPCases(t *testing.T, cases iter.Seq[buildHTTPCase]) {
	t.Helper()
	for tc := range cases {
		t.Run(tc.name, runBuildHTTPCase(tc))
	}
}

func runBuildHTTPCase(tc buildHTTPCase) func(*testing.T) {
	return func(t *testing.T) {
		method := tc.method
		if method == "" {
			method = http.MethodPost
		}
		writer := tc.writer
		if writer == nil {
			writer = noopWriter()
		}
		producers := tc.producers
		if producers == nil {
			producers = New("example.com", "/", []string{schemeHTTP}).Producers
		}

		r := newRequest(method, "/", writer)
		r.consumes = tc.consumes
		req, err := r.BuildHTTP(tc.mediaType, "/", producers, strfmt.Default, nil)
		if tc.wantErr != "" {
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
			return
		}
		require.NoError(t, err)

		got := req.Header.Get(runtime.HeaderContentType)
		assertContentType(t, tc.wantContentType, tc.wantContentTypePrefix, got)
		if tc.wantBody != nil {
			body := readBody(t, req)
			tc.wantBody(t, body)
		}
	}
}

func runSubmitCases(t *testing.T, cases iter.Seq[submitCase]) {
	t.Helper()
	for tc := range cases {
		t.Run(tc.name, runSubmitCase(tc))
	}
}

func runSubmitCase(tc submitCase) func(*testing.T) {
	return func(t *testing.T) {
		method := tc.method
		if method == "" {
			method = http.MethodPost
		}
		writer := tc.writer
		if writer == nil {
			writer = noopWriter()
		}

		var captured *http.Request
		var capturedBody []byte
		transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			captured = req
			if req.Body != nil {
				b, _ := io.ReadAll(req.Body)
				capturedBody = b
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewReader(nil)),
				Request:    req,
			}, nil
		})

		rt := New("example.com", "/", []string{schemeHTTP})
		rt.Transport = transport
		if tc.producers != nil {
			rt.Producers = tc.producers
		}

		_, err := rt.Submit(&runtime.ClientOperation{
			ID:                 "test",
			Method:             method,
			PathPattern:        "/",
			ProducesMediaTypes: tc.producesAccept,
			ConsumesMediaTypes: tc.consumes,
			Params:             writer,
			Reader: runtime.ClientResponseReaderFunc(func(_ runtime.ClientResponse, _ runtime.Consumer) (any, error) {
				return nil, nil
			}),
			Context: context.Background(),
		})
		if tc.wantErr != "" {
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
			return
		}
		require.NoError(t, err)
		require.NotNil(t, captured, "RoundTripper not invoked")

		got := captured.Header.Get(runtime.HeaderContentType)
		assertContentType(t, tc.wantContentType, tc.wantContentTypePrefix, got)
		if tc.wantBody != nil {
			tc.wantBody(t, capturedBody)
		}
	}
}

// assertContentType applies whichever Content-Type expectation the case
// declares: exact, prefix (for multipart), or "no header expected" when
// both are empty.
func assertContentType(t *testing.T, wantExact, wantPrefix, got string) {
	t.Helper()
	if wantPrefix != "" {
		require.True(t, strings.HasPrefix(got, wantPrefix),
			"want Content-Type prefix %q, got %q", wantPrefix, got)
		return
	}
	assert.EqualT(t, wantExact, got)
}

// readBody reads req.Body fully into a buffer, restoring it as a fresh
// reader so the caller can re-read if needed.
func readBody(t *testing.T, req *http.Request) []byte {
	t.Helper()
	if req.Body == nil {
		return nil
	}
	b, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	req.Body = io.NopCloser(bytes.NewReader(b))
	return b
}

// --- writer helpers --------------------------------------------------

func noopWriter() runtime.ClientRequestWriter {
	return runtime.ClientRequestWriterFunc(func(_ runtime.ClientRequest, _ strfmt.Registry) error {
		return nil
	})
}

func writerSetBody(payload any) runtime.ClientRequestWriter {
	return runtime.ClientRequestWriterFunc(func(r runtime.ClientRequest, _ strfmt.Registry) error {
		return r.SetBodyParam(payload)
	})
}

func writerSetHeader(name, value string) runtime.ClientRequestWriter {
	return runtime.ClientRequestWriterFunc(func(r runtime.ClientRequest, _ strfmt.Registry) error {
		return r.SetHeaderParam(name, value)
	})
}

func writerSetForm(name string, values ...string) runtime.ClientRequestWriter {
	return runtime.ClientRequestWriterFunc(func(r runtime.ClientRequest, _ strfmt.Registry) error {
		return r.SetFormParam(name, values...)
	})
}

func writerSetFile(name string, files ...runtime.NamedReadCloser) runtime.ClientRequestWriter {
	return runtime.ClientRequestWriterFunc(func(r runtime.ClientRequest, _ strfmt.Registry) error {
		return r.SetFileParam(name, files...)
	})
}

func writerCompose(writers ...runtime.ClientRequestWriter) runtime.ClientRequestWriter {
	return runtime.ClientRequestWriterFunc(func(r runtime.ClientRequest, reg strfmt.Registry) error {
		for _, w := range writers {
			if err := w.WriteToRequest(r, reg); err != nil {
				return err
			}
		}
		return nil
	})
}

// readerWithCT wraps an io.Reader and declares its MIME type via the
// `ContentType() string` opt-in interface that buildHTTP consults for
// stream payloads.
type readerWithCT struct {
	io.Reader

	ct string
}

func (r *readerWithCT) ContentType() string { return r.ct }

// readCloserWithCT is the ReadCloser counterpart.
type readCloserWithCT struct {
	io.ReadCloser

	ct string
}

func (r *readCloserWithCT) ContentType() string { return r.ct }

// staticFile is a NamedReadCloser used in file-upload cases.
type staticFile struct {
	name string
	r    *strings.Reader
	ct   string // optional ContentType()
}

func (f *staticFile) Name() string               { return f.name }
func (f *staticFile) Read(p []byte) (int, error) { return f.r.Read(p) }
func (f *staticFile) Close() error               { return nil }

// staticFileWithCT implements NamedReadCloser AND ContentType() string.
type staticFileWithCT struct{ *staticFile }

func (f *staticFileWithCT) ContentType() string { return f.ct }

func newFile(name, data string) runtime.NamedReadCloser {
	return &staticFile{name: name, r: strings.NewReader(data)}
}

func newFileWithCT(name, data, ct string) runtime.NamedReadCloser {
	return &staticFileWithCT{staticFile: &staticFile{name: name, r: strings.NewReader(data), ct: ct}}
}

// --- body assertion helpers ------------------------------------------

func bodyExact(want string) func(t *testing.T, body []byte) {
	return func(t *testing.T, body []byte) {
		t.Helper()
		assert.EqualT(t, want, string(body))
	}
}

func bodyContainsAll(parts ...string) func(t *testing.T, body []byte) {
	return func(t *testing.T, body []byte) {
		t.Helper()
		s := string(body)
		for _, p := range parts {
			assert.Contains(t, s, p, "expected body to contain %q", p)
		}
	}
}

func bodyEmpty() func(t *testing.T, body []byte) {
	return func(t *testing.T, body []byte) {
		t.Helper()
		assert.Empty(t, body)
	}
}

// --- test entry points ----------------------------------------------

func TestBuildHTTP_ContentNegotiation(t *testing.T) {
	runBuildHTTPCases(t, payloadStructCases())
	runBuildHTTPCases(t, payloadReaderCases())
	runBuildHTTPCases(t, payloadByteSliceCases())
	runBuildHTTPCases(t, formFieldCases())
	runBuildHTTPCases(t, fileFieldCases())
	runBuildHTTPCases(t, formAndFileFieldCases())
	runBuildHTTPCases(t, noBodyCases())
	runBuildHTTPCases(t, missingProducerCases())
}

func TestSubmit_ContentNegotiation(t *testing.T) {
	runSubmitCases(t, submitWiringCases())
}

// --- case families ---------------------------------------------------

// Family A — payload as struct, producer runs.
func payloadStructCases() iter.Seq[buildHTTPCase] {
	return slices.Values([]buildHTTPCase{
		{
			name:            "struct + JSON producer",
			mediaType:       runtime.JSONMime,
			writer:          writerSetBody(task{Completed: true, Content: "ok", ID: 7}),
			wantContentType: runtime.JSONMime,
			wantBody:        bodyContainsAll(`"completed":true`, `"content":"ok"`, `"id":7`),
		},
		{
			name:            "struct + XML producer",
			mediaType:       runtime.XMLMime,
			writer:          writerSetBody(task{Completed: false, Content: "x", ID: 1}),
			wantContentType: runtime.XMLMime,
			wantBody:        bodyContainsAll("<task>", "<content>x</content>"),
		},
		{
			name:            "struct + text producer",
			mediaType:       runtime.TextMime,
			writer:          writerSetBody("a-string-payload"),
			wantContentType: runtime.TextMime,
			wantBody:        bodyExact("a-string-payload"),
		},
		{
			// Deliberate non-honor: SetHeader is NOT respected for
			// struct payloads because the producer is dispatched off
			// mediaType. Honouring an arbitrary user header here would
			// mean either swapping the producer (complex) or sending a
			// body that doesn't match the declared header (still a
			// lie). Streams have no producer dispatch, so they get the
			// escape hatch; struct/form/multipart paths do not.
			name:      "struct + SetHeader Content-Type is ignored — picker wins",
			mediaType: runtime.JSONMime,
			writer: writerCompose(
				writerSetHeader("Content-Type", "application/x-ignored"),
				writerSetBody(task{Content: "y"}),
			),
			wantContentType: runtime.JSONMime,
			wantBody:        bodyContainsAll(`"content":"y"`),
		},
		// Note: missing-producer for non-Reader payload panics today
		// inside buildHTTP (nil producer dereference at request.go:343).
		// The Submit-level gate at runtime.go:550 catches it earlier and
		// returns a proper error — covered by submitWiringCases.
	})
}

// Family B — payload as io.Reader / io.ReadCloser. The producer is
// bypassed; the body is the reader bytes verbatim.
//
// Header rules (in priority order):
//  1. payload's `ContentType() string` if non-empty;
//  2. application/octet-stream from the consumes list, when registered
//     as a producer (Stage-2 fallback);
//  3. the picker's mediaType.
//
// Cases with empty consumes exercise the buildHTTP-direct entry point
// (i.e. external callers of BuildHTTP that have already picked a mime
// without going through createHttpRequest).
func payloadReaderCases() iter.Seq[buildHTTPCase] {
	return slices.Values([]buildHTTPCase{
		{
			name:            "io.Reader, picked octet-stream — header matches body",
			mediaType:       runtime.DefaultMime,
			writer:          writerSetBody(bytes.NewReader([]byte{0x01, 0x02, 0x03})),
			wantContentType: runtime.DefaultMime,
			wantBody:        bodyExact("\x01\x02\x03"),
		},
		{
			// No consumes context: external BuildHTTP caller already
			// picked JSON. Header is the picked mime even though the
			// body is raw bytes — we have nothing better to fall back
			// to.
			name:            "io.Reader, picked JSON, no consumes context — header is picked mime",
			mediaType:       runtime.JSONMime,
			writer:          writerSetBody(bytes.NewReader([]byte("not json at all"))),
			wantContentType: runtime.JSONMime,
			wantBody:        bodyExact("not json at all"),
		},
		{
			// Stage-2 kicks in: picker chose JSON, but octet-stream is
			// also offered and registered, so the wire header drops the
			// JSON claim and advertises raw bytes instead.
			name:            "io.Reader, picked JSON, octet-stream offered → octet-stream wins",
			mediaType:       runtime.JSONMime,
			consumes:        []string{runtime.JSONMime, runtime.DefaultMime},
			writer:          writerSetBody(bytes.NewReader([]byte("not json"))),
			wantContentType: runtime.DefaultMime,
			wantBody:        bodyExact("not json"),
		},
		{
			// Stage-2 has nothing to upgrade to: octet-stream is not
			// among the candidates, so the picker's choice is preserved.
			name:            "io.Reader, picked text, no octet-stream offered → picked mime preserved",
			mediaType:       runtime.TextMime,
			consumes:        []string{runtime.TextMime, runtime.JSONMime},
			writer:          writerSetBody(bytes.NewReader([]byte("hi"))),
			wantContentType: runtime.TextMime,
			wantBody:        bodyExact("hi"),
		},
		{
			name:      "io.Reader with ContentType() overrides Stage-2 fallback",
			mediaType: runtime.JSONMime,
			consumes:  []string{runtime.JSONMime, runtime.DefaultMime},
			writer: writerSetBody(&readerWithCT{
				Reader: strings.NewReader(`{"a":1}` + "\n" + `{"a":2}`),
				ct:     ndjsonMime,
			}),
			wantContentType: ndjsonMime,
			wantBody:        bodyExact("{\"a\":1}\n{\"a\":2}"),
		},
		{
			name:      "io.Reader with empty ContentType() — Stage-2 fallback applies",
			mediaType: runtime.JSONMime,
			consumes:  []string{runtime.JSONMime, runtime.DefaultMime},
			writer: writerSetBody(&readerWithCT{
				Reader: strings.NewReader("body"),
				ct:     "",
			}),
			wantContentType: runtime.DefaultMime,
			wantBody:        bodyExact("body"),
		},
		{
			name:            "io.ReadCloser, picked octet-stream",
			mediaType:       runtime.DefaultMime,
			writer:          writerSetBody(io.NopCloser(strings.NewReader("payload"))),
			wantContentType: runtime.DefaultMime,
			wantBody:        bodyExact("payload"),
		},
		{
			name:            "io.ReadCloser, picked text, octet-stream offered → octet-stream wins",
			mediaType:       runtime.TextMime,
			consumes:        []string{runtime.TextMime, runtime.DefaultMime},
			writer:          writerSetBody(io.NopCloser(bytes.NewReader([]byte{0xFF, 0xFE}))),
			wantContentType: runtime.DefaultMime,
			wantBody:        bodyExact("\xFF\xFE"),
		},
		{
			name:      "io.ReadCloser with ContentType() overrides Stage-2 fallback",
			mediaType: runtime.TextMime,
			consumes:  []string{runtime.TextMime, runtime.DefaultMime},
			writer: writerSetBody(&readCloserWithCT{
				ReadCloser: io.NopCloser(strings.NewReader("ndjson")),
				ct:         ndjsonMime,
			}),
			wantContentType: ndjsonMime,
			wantBody:        bodyExact("ndjson"),
		},
		{
			// Edge case: octet-stream offered but no producer registered
			// for it (caller stripped the default). Stage-2 cannot
			// upgrade, picker's choice preserved.
			name:      "io.Reader, octet-stream offered but producer missing → picked mime preserved",
			mediaType: runtime.JSONMime,
			consumes:  []string{runtime.JSONMime, runtime.DefaultMime},
			producers: map[string]runtime.Producer{
				runtime.JSONMime: runtime.JSONProducer(),
			},
			writer:          writerSetBody(bytes.NewReader([]byte("x"))),
			wantContentType: runtime.JSONMime,
			wantBody:        bodyExact("x"),
		},
		{
			// Highest-priority escape hatch: SetHeaderParam wins over
			// every derivation. Picker chose JSON, payload is a plain
			// Reader, but the user asserted application/x-vendor.
			name:      "io.Reader + SetHeader Content-Type wins over picker",
			mediaType: runtime.JSONMime,
			writer: writerCompose(
				writerSetHeader("Content-Type", vendorMime),
				writerSetBody(bytes.NewReader([]byte("v"))),
			),
			wantContentType: vendorMime,
			wantBody:        bodyExact("v"),
		},
		{
			// SetHeader beats payload's ContentType() declaration.
			name:      "io.Reader + SetHeader wins over payload ContentType()",
			mediaType: runtime.JSONMime,
			writer: writerCompose(
				writerSetHeader("Content-Type", "application/x-explicit"),
				writerSetBody(&readerWithCT{
					Reader: strings.NewReader("body"),
					ct:     ndjsonMime,
				}),
			),
			wantContentType: "application/x-explicit",
			wantBody:        bodyExact("body"),
		},
		{
			// SetHeader beats Stage-2 octet-stream upgrade too.
			name:      "io.Reader + SetHeader wins over Stage-2 octet-stream",
			mediaType: runtime.JSONMime,
			consumes:  []string{runtime.JSONMime, runtime.DefaultMime},
			writer: writerCompose(
				writerSetHeader("Content-Type", "application/x-explicit"),
				writerSetBody(bytes.NewReader([]byte("v"))),
			),
			wantContentType: "application/x-explicit",
			wantBody:        bodyExact("v"),
		},
		{
			// io.ReadCloser parity.
			name:      "io.ReadCloser + SetHeader Content-Type wins",
			mediaType: runtime.TextMime,
			writer: writerCompose(
				writerSetHeader("Content-Type", vendorMime),
				writerSetBody(io.NopCloser(strings.NewReader("data"))),
			),
			wantContentType: vendorMime,
			wantBody:        bodyExact("data"),
		},
	})
}

// Family C — []byte payload (producer runs, encoding the slice).
//
// Note: []byte does not satisfy io.Reader, so it falls through to the
// producer. The JSON producer base64-encodes []byte per
// encoding/json's documented behaviour.
func payloadByteSliceCases() iter.Seq[buildHTTPCase] {
	return slices.Values([]buildHTTPCase{
		{
			name:            "[]byte + JSON producer (base64-encoded as string)",
			mediaType:       runtime.JSONMime,
			writer:          writerSetBody([]byte("hello")),
			wantContentType: runtime.JSONMime,
			// "hello" base64 = aGVsbG8=
			wantBody: bodyContainsAll("aGVsbG8="),
		},
		{
			name:            "[]byte + ByteStreamProducer (raw bytes)",
			mediaType:       runtime.DefaultMime,
			writer:          writerSetBody([]byte("hello")),
			wantContentType: runtime.DefaultMime,
			wantBody:        bodyExact("hello"),
		},
	})
}

// Family D — form fields only. The buildHTTP form path runs whenever
// formFields > 0 OR fileFields > 0. mediaType drives the multipart
// vs urlencoded choice via isMultipart.
func formFieldCases() iter.Seq[buildHTTPCase] {
	return slices.Values([]buildHTTPCase{
		{
			name:            "form fields + urlencoded mime",
			mediaType:       runtime.URLencodedFormMime,
			writer:          writerCompose(writerSetForm("name", "fido"), writerSetForm("color", "brown")),
			wantContentType: runtime.URLencodedFormMime,
			wantBody:        bodyContainsAll("name=fido", "color=brown"),
		},
		{
			name:                  "form fields + multipart mime",
			mediaType:             runtime.MultipartFormMime,
			writer:                writerSetForm("name", "fido"),
			wantContentTypePrefix: runtime.MultipartFormMime + "; boundary=",
			wantBody:              bodyContainsAll(`name="name"`, "fido"),
		},
		{
			name:            "form fields + non-form mime — header lies",
			mediaType:       runtime.JSONMime,
			writer:          writerSetForm("name", "fido"),
			wantContentType: runtime.JSONMime,
			wantBody:        bodyContainsAll("name=fido"),
		},
	})
}

// Family E — file fields only.
func fileFieldCases() iter.Seq[buildHTTPCase] {
	return slices.Values([]buildHTTPCase{
		{
			name:                  "file field + multipart mime",
			mediaType:             runtime.MultipartFormMime,
			writer:                writerSetFile("upload", newFile("doc.txt", "filebody")),
			wantContentTypePrefix: runtime.MultipartFormMime + "; boundary=",
			wantBody:              bodyContainsAll(`name="upload"`, `filename="doc.txt"`, "filebody"),
		},
		{
			// Per the post-#286 fix, urlencoded with files is allowed and
			// the file content travels as a regular form value.
			name:            "file field + urlencoded mime — file inlined as form value",
			mediaType:       runtime.URLencodedFormMime,
			writer:          writerSetFile("upload", newFile("doc.txt", "abc")),
			wantContentType: runtime.URLencodedFormMime,
			wantBody:        bodyContainsAll("upload=abc"),
		},
		{
			name:                  "file with declared ContentType()",
			mediaType:             runtime.MultipartFormMime,
			writer:                writerSetFile("upload", newFileWithCT("doc.txt", "x", "application/json")),
			wantContentTypePrefix: runtime.MultipartFormMime + "; boundary=",
			wantBody:              bodyContainsAll("application/json"),
		},
	})
}

// Family F — form + file fields. Multipart preferred (post-#286).
func formAndFileFieldCases() iter.Seq[buildHTTPCase] {
	return slices.Values([]buildHTTPCase{
		{
			name:      "form + file + multipart mime",
			mediaType: runtime.MultipartFormMime,
			writer: writerCompose(
				writerSetForm("name", "fido"),
				writerSetFile("upload", newFile("doc.txt", "filebody")),
			),
			wantContentTypePrefix: runtime.MultipartFormMime + "; boundary=",
			wantBody:              bodyContainsAll(`name="name"`, "fido", `filename="doc.txt"`, "filebody"),
		},
		{
			name:      "form + file + urlencoded mime — file inlined",
			mediaType: runtime.URLencodedFormMime,
			writer: writerCompose(
				writerSetForm("name", "fido"),
				writerSetFile("upload", newFile("doc.txt", "abc")),
			),
			wantContentType: runtime.URLencodedFormMime,
			wantBody:        bodyContainsAll("name=fido", "upload=abc"),
		},
	})
}

// Family G — no body.
func noBodyCases() iter.Seq[buildHTTPCase] {
	return slices.Values([]buildHTTPCase{
		{
			name:            "GET, no payload, no fields — no Content-Type",
			method:          http.MethodGet,
			mediaType:       runtime.JSONMime, // mediaType is ignored when there is no body
			writer:          noopWriter(),
			wantContentType: "",
			wantBody:        bodyEmpty(),
		},
		{
			name:            "POST, no payload, no fields — no Content-Type",
			method:          http.MethodPost,
			mediaType:       runtime.JSONMime,
			writer:          noopWriter(),
			wantContentType: "",
			wantBody:        bodyEmpty(),
		},
	})
}

// Family — error paths in buildHTTP / write phase.
func missingProducerCases() iter.Seq[buildHTTPCase] {
	// buildHTTP looks up the producer at line 342 only for non-Reader
	// payloads. A nil producer panics there today, so we cannot easily
	// assert via this harness without recovering. The Submit-level
	// equivalent is gated earlier (runtime.go:550) and returns a
	// proper error — covered in submitWiringCases.
	return slices.Values([]buildHTTPCase{
		{
			name:      "writer returns an error",
			mediaType: runtime.JSONMime,
			writer: runtime.ClientRequestWriterFunc(func(_ runtime.ClientRequest, _ strfmt.Registry) error {
				return errors.New("boom")
			}),
			wantErr: "boom",
		},
	})
}

// Family H — Submit-level: verifies picker → buildHTTP wiring through
// a captured RoundTripper.
func submitWiringCases() iter.Seq[submitCase] {
	return slices.Values([]submitCase{
		{
			name:            "consumes [json] + struct payload",
			consumes:        []string{runtime.JSONMime},
			writer:          writerSetBody(task{Content: "x"}),
			wantContentType: runtime.JSONMime,
			wantBody:        bodyContainsAll(`"content":"x"`),
		},
		{
			name:                  "consumes [multipart, urlencoded] + form fields → multipart wins",
			consumes:              []string{runtime.URLencodedFormMime, runtime.MultipartFormMime},
			writer:                writerSetForm("name", "fido"),
			wantContentTypePrefix: runtime.MultipartFormMime + "; boundary=",
			wantBody:              bodyContainsAll(`name="name"`, "fido"),
		},
		{
			// Stage-2 fix: picker chose JSON (first non-empty), but
			// the payload is a stream and octet-stream is offered —
			// so the wire header is upgraded to octet-stream.
			name:            "consumes [json, octet] + io.Reader → octet-stream wins (Stage-2)",
			consumes:        []string{runtime.JSONMime, runtime.DefaultMime},
			writer:          writerSetBody(bytes.NewReader([]byte("not json"))),
			wantContentType: runtime.DefaultMime,
			wantBody:        bodyExact("not json"),
		},
		{
			// Picker chose JSON; no octet-stream offered. Stage-2 has
			// nothing to upgrade to, so the wire header is JSON even
			// though the body is raw bytes.
			name:            "consumes [json] + io.Reader without alternative → header is picked JSON",
			consumes:        []string{runtime.JSONMime},
			writer:          writerSetBody(bytes.NewReader([]byte("raw"))),
			wantContentType: runtime.JSONMime,
			wantBody:        bodyExact("raw"),
		},
		{
			// SetHeader escape hatch surfaces through Submit too.
			name:     "consumes [json] + SetHeader Content-Type — escape hatch wins",
			consumes: []string{runtime.JSONMime},
			writer: writerCompose(
				writerSetHeader("Content-Type", vendorMime),
				writerSetBody(bytes.NewReader([]byte("data"))),
			),
			wantContentType: vendorMime,
			wantBody:        bodyExact("data"),
		},
		{
			name:     "consumes [json] + io.Reader with ContentType() — declared type wins",
			consumes: []string{runtime.JSONMime},
			writer: writerSetBody(&readerWithCT{
				Reader: strings.NewReader(`{"line":1}` + "\n" + `{"line":2}`),
				ct:     ndjsonMime,
			}),
			wantContentType: ndjsonMime,
			wantBody:        bodyExact("{\"line\":1}\n{\"line\":2}"),
		},
		{
			name:     "consumes lists only an unregistered producer — error before send",
			consumes: []string{"application/vnd.example"},
			writer:   writerSetBody(task{Content: "x"}),
			wantErr:  "none of producers",
		},
		{
			// Producer-capability filter: spec lists a vendor mime first
			// but no vendor producer is registered. Picker now falls
			// through to the registered JSON entry instead of erroring.
			// Resolves issues #32, #386 (filter rule).
			name:            "consumes [vendor, json] with no vendor producer → JSON wins",
			consumes:        []string{"application/x-vendor", runtime.JSONMime},
			writer:          writerSetBody(task{Content: "x"}),
			wantContentType: runtime.JSONMime,
			wantBody:        bodyContainsAll(`"content":"x"`),
		},
		{
			// All entries unregistered: picker preserves the first
			// non-empty so the runtime.go gate fires with the historical
			// diagnostic.
			name:     "consumes lists only unregistered producers → error before send",
			consumes: []string{vendorMime1, vendorMime2},
			writer:   writerSetBody(task{Content: "x"}),
			wantErr:  "none of producers",
		},
		{
			name:            "empty consumes falls back to DefaultMediaType (json)",
			consumes:        nil,
			writer:          writerSetBody(task{Content: "x"}),
			wantContentType: runtime.JSONMime,
			wantBody:        bodyContainsAll(`"content":"x"`),
		},
	})
}
