// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

var _ runtime.ClientRequest = new(request) // ensure compliance to the interface

// Request represents a swagger client request.
//
// This Request struct converts to a HTTP request.
// There might be others that convert to other transports.
// There is no error checking here, it is assumed to be used after a spec has been validated.
// so impossible combinations should not arise (hopefully).
//
// The main purpose of this struct is to hide the machinery of adding params to a transport request.
// The generated code only implements what is necessary to turn a param into a valid value for these methods.
type request struct {
	pathPattern string
	method      string
	writer      runtime.ClientRequestWriter

	pathParams map[string]string
	header     http.Header
	query      url.Values
	formFields url.Values
	fileFields map[string][]runtime.NamedReadCloser
	payload    any
	// consumes carries the operation's full ConsumesMediaTypes list so
	// that buildHTTP — which runs after the writer populates the payload
	// — can apply payload-aware fallback rules (see streamFallbackMime).
	// Set by Runtime.createHttpRequest. Direct buildHTTP callers leave it
	// nil and get unchanged single-mime behaviour.
	consumes []string
	timeout  time.Duration
	buf      *bytes.Buffer

	getBody func(r *request) []byte
}

// NewRequest creates a new swagger http client request.
func newRequest(method, pathPattern string, writer runtime.ClientRequestWriter) *request {
	return &request{
		pathPattern: pathPattern,
		method:      method,
		writer:      writer,
		header:      make(http.Header),
		query:       make(url.Values),
		timeout:     DefaultTimeout,
		getBody:     getRequestBuffer,
	}
}

// BuildHTTP creates a new http request based on the data from the params.
func (r *request) BuildHTTP(mediaType, basePath string, producers map[string]runtime.Producer, registry strfmt.Registry) (*http.Request, error) {
	return r.buildHTTP(mediaType, basePath, producers, registry, nil)
}

func (r *request) GetMethod() string {
	return r.method
}

func (r *request) GetPath() string {
	path := r.pathPattern
	for k, v := range r.pathParams {
		path = strings.ReplaceAll(path, "{"+k+"}", v)
	}
	return path
}

func (r *request) GetBody() []byte {
	return r.getBody(r)
}

// SetHeaderParam adds a header param to the request
// when there is only 1 value provided for the varargs, it will set it.
// when there are several values provided for the varargs it will add it (no overriding).
func (r *request) SetHeaderParam(name string, values ...string) error {
	if r.header == nil {
		r.header = make(http.Header)
	}
	r.header[http.CanonicalHeaderKey(name)] = values
	return nil
}

// GetHeaderParams returns the all headers currently set for the request.
func (r *request) GetHeaderParams() http.Header {
	return r.header
}

// SetQueryParam adds a query param to the request
// when there is only 1 value provided for the varargs, it will set it.
// when there are several values provided for the varargs it will add it (no overriding).
func (r *request) SetQueryParam(name string, values ...string) error {
	if r.query == nil {
		r.query = make(url.Values)
	}
	r.query[name] = values
	return nil
}

// GetQueryParams returns a copy of all query params currently set for the request.
func (r *request) GetQueryParams() url.Values {
	var result = make(url.Values)
	for key, value := range r.query {
		result[key] = append([]string{}, value...)
	}
	return result
}

// SetFormParam adds a forn param to the request
// when there is only 1 value provided for the varargs, it will set it.
// when there are several values provided for the varargs it will add it (no overriding).
func (r *request) SetFormParam(name string, values ...string) error {
	if r.formFields == nil {
		r.formFields = make(url.Values)
	}
	r.formFields[name] = values
	return nil
}

// SetPathParam adds a path param to the request.
func (r *request) SetPathParam(name string, value string) error {
	if r.pathParams == nil {
		r.pathParams = make(map[string]string)
	}

	r.pathParams[name] = value
	return nil
}

// SetFileParam adds a file param to the request.
func (r *request) SetFileParam(name string, files ...runtime.NamedReadCloser) error {
	for _, file := range files {
		if actualFile, ok := file.(*os.File); ok {
			fi, err := os.Stat(actualFile.Name())
			if err != nil {
				return err
			}
			if fi.IsDir() {
				return fmt.Errorf("%q is a directory, only files are supported", file.Name())
			}
		}
	}

	if r.fileFields == nil {
		r.fileFields = make(map[string][]runtime.NamedReadCloser)
	}
	if r.formFields == nil {
		r.formFields = make(url.Values)
	}

	r.fileFields[name] = files
	return nil
}

func (r *request) GetFileParam() map[string][]runtime.NamedReadCloser {
	return r.fileFields
}

// SetBodyParam sets a body parameter on the request.
// This does not yet serialze the object, this happens as late as possible.
func (r *request) SetBodyParam(payload any) error {
	r.payload = payload
	return nil
}

func (r *request) GetBodyParam() any {
	return r.payload
}

// SetTimeout sets the timeout for a request.
func (r *request) SetTimeout(timeout time.Duration) error {
	r.timeout = timeout
	return nil
}

func (r *request) isMultipart(mediaType string) bool {
	// An explicit application/x-www-form-urlencoded choice is honored even when
	// file fields are present: the spec allows files to travel as URL-encoded
	// form values, although it does not stream and is discouraged. Without this
	// short-circuit, picking urlencoded with files would silently fall back to
	// multipart and emit an inconsistent Content-Type.
	if strings.EqualFold(mediaType, runtime.URLencodedFormMime) {
		return false
	}
	if len(r.fileFields) > 0 {
		return true
	}

	return runtime.MultipartFormMime == mediaType
}

func (r *request) buildHTTP(mediaType, basePath string, producers map[string]runtime.Producer, registry strfmt.Registry, auth runtime.ClientAuthInfoWriter) (*http.Request, error) {
	// build the data
	if err := r.writer.WriteToRequest(r, registry); err != nil {
		return nil, err
	}

	body, err := r.buildBody(mediaType, producers)
	if err != nil {
		return nil, err
	}

	if runtime.CanHaveBody(r.method) && body != nil && r.header.Get(runtime.HeaderContentType) == "" {
		r.header.Set(runtime.HeaderContentType, mediaType)
	}

	body, err = r.applyAuth(auth, body, registry)
	if err != nil {
		return nil, err
	}

	urlPath, staticQueryParams, err := r.resolveURLPath(basePath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(context.Background(), r.method, urlPath, body)
	if err != nil {
		return nil, err
	}

	if err := r.mergeStaticQuery(staticQueryParams); err != nil {
		return nil, err
	}

	req.URL.RawQuery = r.query.Encode()
	req.Header = r.header

	return req, nil
}

// resolveURLPath builds the final url path string and returns the static
// query parameters extracted from basePath and r.pathPattern.
//
// Static query parameters from the path pattern take precedence over those
// from the base path; merging with r.query is the caller's responsibility
// (see [request.mergeStaticQuery]).
//
// The path is assembled from basePath + pathPattern with path-param
// substitution and trailing-slash preservation when the original
// pathPattern carried one.
func (r *request) resolveURLPath(basePath string) (string, url.Values, error) {
	basePathURL, err := url.Parse(basePath)
	if err != nil {
		return "", nil, err
	}
	staticQueryParams := basePathURL.Query()

	pathPatternURL, err := url.Parse(r.pathPattern)
	if err != nil {
		return "", nil, err
	}
	for name, values := range pathPatternURL.Query() {
		if _, present := staticQueryParams[name]; present {
			staticQueryParams.Del(name)
		}
		for _, value := range values {
			staticQueryParams.Add(name, value)
		}
	}

	reinstateSlash := pathPatternURL.Path != "" && pathPatternURL.Path != "/" &&
		pathPatternURL.Path[len(pathPatternURL.Path)-1] == '/'

	urlPath := path.Join(basePathURL.Path, pathPatternURL.Path)
	for k, v := range r.pathParams {
		urlPath = strings.ReplaceAll(urlPath, "{"+k+"}", url.PathEscape(v))
	}
	if reinstateSlash {
		urlPath += "/"
	}

	return urlPath, staticQueryParams, nil
}

// applyAuth runs auth.AuthenticateRequest with a lazy getBody closure
// installed for cases where the http.Request body is not r.buf — i.e.
// the payload is an io.Reader / io.ReadCloser, or we are doing a
// multipart form/file upload.
//
// In those cases, if AuthenticateRequest asks for the body, we copy
// the stream/pipe into r.buf on demand and provide that. The closure
// also reassigns the local body to r.buf so the post-auth body source
// passed to http.NewRequestWithContext is the buffered copy.
//
// The closure is registered lazily because there is no way to know
// ahead of time whether AuthenticateRequest will read the body.
//
// On error precedence: a copy error is reported in preference to the
// AuthenticateRequest error, because a mis-read body may have
// interfered with auth.
//
// No-op when auth is nil; returns body unchanged.
func (r *request) applyAuth(auth runtime.ClientAuthInfoWriter, body io.Reader, registry strfmt.Registry) (io.Reader, error) {
	if auth == nil {
		return body, nil
	}

	var copyErr error
	if buf, ok := body.(*bytes.Buffer); body != nil && (!ok || buf != r.buf) {
		var copied bool
		r.getBody = func(r *request) []byte {
			if copied {
				return getRequestBuffer(r)
			}

			defer func() {
				copied = true
			}()

			if _, copyErr = io.Copy(r.buf, body); copyErr != nil {
				return nil
			}

			if closer, ok := body.(io.ReadCloser); ok {
				if copyErr = closer.Close(); copyErr != nil {
					return nil
				}
			}

			body = r.buf
			return getRequestBuffer(r)
		}
	}

	authErr := auth.AuthenticateRequest(r, registry)

	if copyErr != nil {
		return nil, fmt.Errorf("error retrieving the response body: %v", copyErr)
	}

	if authErr != nil {
		return nil, authErr
	}

	return body, nil
}

// mergeStaticQuery overlays staticQuery onto r.query. On conflict r.query
// wins — the parameters set by the client take precedence over the ones
// extracted from basePath / pathPattern.
func (r *request) mergeStaticQuery(staticQuery url.Values) error {
	originalParams := r.GetQueryParams()
	for k, v := range staticQuery {
		if _, present := originalParams[k]; present {
			continue
		}
		if err := r.SetQueryParam(k, v...); err != nil {
			return err
		}
	}
	return nil
}

// buildBody dispatches to the appropriate body-construction helper based
// on what the operation has populated. Initializes r.buf as the working
// buffer (used by helpers that buffer their output, and later read back
// by getRequestBuffer for auth body access).
//
// Returns (nil, nil) when the request carries no body — neither a
// payload nor any form/file fields.
//
// Each helper sets the Content-Type header itself; this function does
// not.
func (r *request) buildBody(mediaType string, producers map[string]runtime.Producer) (io.Reader, error) {
	r.buf = bytes.NewBuffer(nil)

	switch {
	case len(r.formFields) > 0 || len(r.fileFields) > 0:
		if r.isMultipart(mediaType) {
			return r.writeMultipartBody(mediaType), nil
		}
		return r.writeURLEncodedBody(mediaType)
	case r.payload != nil:
		return r.writePayloadBody(mediaType, producers)
	}

	// nilnil: nil body / nil error means "no body to send" — the caller
	// distinguishes this from an error via `body != nil`. Introducing a
	// sentinel error would force the caller to compare against it before
	// every error check, which is more complex than the current shape.
	return nil, nil //nolint:nilnil
}

// writeURLEncodedBody serializes form fields (and any file fields, per
// Swagger 2.0 fallback semantics) into r.buf as
// application/x-www-form-urlencoded. Sets Content-Type to mediaType and
// returns r.buf as the body source.
//
// Per Swagger 2.0, file form parameters can be sent under
// application/x-www-form-urlencoded by including the file content as a
// regular form-field value. The whole form is then percent-encoded as
// usual. This buffers the entire payload and does not preserve a
// per-file Content-Type — multipart/form-data is preferred when both
// are advertised by the operation.
func (r *request) writeURLEncodedBody(mediaType string) (io.Reader, error) {
	r.header.Set(runtime.HeaderContentType, mediaType)
	values := url.Values{}
	for k, vs := range r.formFields {
		values[k] = append(values[k], vs...)
	}
	for fn, ff := range r.fileFields {
		for _, fi := range ff {
			data, ferr := io.ReadAll(fi)
			if cerr := fi.Close(); cerr != nil && ferr == nil {
				ferr = cerr
			}
			if ferr != nil {
				return nil, ferr
			}
			values.Add(fn, string(data))
		}
	}
	r.buf.WriteString(values.Encode())
	return r.buf, nil
}

// writeMultipartBody assembles a multipart/form-data body via an
// io.Pipe. A goroutine streams form fields and files into the pipe
// writer; the pipe reader is returned as the body. Sets Content-Type to
// the multipart media type with the writer's boundary parameter.
//
// The goroutine owns the pipe writer's lifecycle: it closes the
// multipart writer (flushing the closing boundary) and the pipe writer
// when it finishes or hits an error.
func (r *request) writeMultipartBody(mediaType string) io.Reader {
	pr, pw := io.Pipe()
	mp := multipart.NewWriter(pw)
	r.header.Set(runtime.HeaderContentType, mangleContentType(mediaType, mp.Boundary()))

	go r.streamMultipartParts(mp, pw)

	return pr
}

// streamMultipartParts writes form fields then file fields to mp,
// closing mp and pw when done. Errors are reported by closing pw with
// the error so the consumer of pr observes them on its next Read.
func (r *request) streamMultipartParts(mp *multipart.Writer, pw *io.PipeWriter) {
	defer func() {
		mp.Close()
		pw.Close()
	}()

	for fn, v := range r.formFields {
		for _, vi := range v {
			if err := mp.WriteField(fn, vi); err != nil {
				logClose(err, pw)
				return
			}
		}
	}

	defer func() {
		for _, ff := range r.fileFields {
			for _, ffi := range ff {
				ffi.Close()
			}
		}
	}()
	for fn, f := range r.fileFields {
		for _, fi := range f {
			var fileContentType string
			if p, ok := fi.(runtime.ContentTyper); ok {
				fileContentType = p.ContentType()
			} else {
				// Need to read the data so that we can detect the content type
				const contentTypeBufferSize = 512
				buf := make([]byte, contentTypeBufferSize)
				size, err := fi.Read(buf)
				if err != nil && err != io.EOF {
					logClose(err, pw)
					return
				}
				fileContentType = http.DetectContentType(buf)
				fi = runtime.NamedReader(fi.Name(), io.MultiReader(bytes.NewReader(buf[:size]), fi))
			}

			// Create the MIME headers for the new part
			h := make(textproto.MIMEHeader)
			h.Set("Content-Disposition",
				fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
					escapeQuotes(fn), escapeQuotes(filepath.Base(fi.Name()))))
			h.Set("Content-Type", fileContentType)

			wrtr, err := mp.CreatePart(h)
			if err != nil {
				logClose(err, pw)
				return
			}
			if _, err := io.Copy(wrtr, fi); err != nil {
				logClose(err, pw)
			}
		}
	}
}

// writePayloadBody handles the r.payload != nil case.
//
// Stream payloads (io.Reader / io.ReadCloser) bypass the producer —
// their bytes flow through verbatim and the wire Content-Type is
// resolved via setStreamContentType (priority: existing header,
// payload's ContentTyper, streamFallbackMime, mediaType).
//
// Non-stream payloads run through the producer registered for
// mediaType. The Content-Type header reflects the picker. Note:
// SetHeaderParam("Content-Type", …) is intentionally NOT honored on
// the producer path because the producer is dispatched off mediaType —
// the wire header would otherwise misrepresent the body. Same
// reasoning applies to the form/multipart branches.
func (r *request) writePayloadBody(mediaType string, producers map[string]runtime.Producer) (io.Reader, error) {
	if rdr, ok := r.payload.(io.ReadCloser); ok {
		setStreamContentType(r.header, r.payload, mediaType, r.consumes, producers)
		return rdr, nil
	}

	if rdr, ok := r.payload.(io.Reader); ok {
		setStreamContentType(r.header, r.payload, mediaType, r.consumes, producers)
		return rdr, nil
	}

	r.header.Set(runtime.HeaderContentType, mediaType)
	producer := producers[mediaType]
	if err := producer.Produce(r.buf, r.payload); err != nil {
		return nil, err
	}
	return r.buf, nil
}

func escapeQuotes(s string) string {
	return strings.NewReplacer("\\", "\\\\", `"`, "\\\"").Replace(s)
}

// setStreamContentType resolves and writes the wire Content-Type for a
// stream payload (io.Reader / io.ReadCloser). Priority:
//
//  1. an explicit value already in header — the user set it via
//     SetHeaderParam during WriteToRequest, and we treat that as an
//     intentional escape hatch;
//  2. payload's [runtime.ContentTyper] declaration;
//  3. [streamFallbackMime] (Stage-2 octet-stream upgrade);
//  4. the picker's mediaType (passed in as the chain's terminal
//     fallback).
//
// Does not apply to non-stream payloads or to form/multipart bodies —
// see the comment above the call site in [request.buildHTTP].
func setStreamContentType(
	header http.Header,
	payload any,
	mediaType string,
	candidates []string,
	producers map[string]runtime.Producer,
) {
	if header.Get(runtime.HeaderContentType) != "" {
		return
	}
	fallback := streamFallbackMime(mediaType, candidates, producers)
	header.Set(runtime.HeaderContentType, payloadContentType(payload, fallback))
}

// payloadContentType returns the payload's declared content type when
// it implements [runtime.ContentTyper] with a non-empty result, and
// fallback otherwise. Mirrors the per-file convention already used for
// multipart upload parts (see [request.buildHTTP] file-fields branch).
func payloadContentType(payload any, fallback string) string {
	if t, ok := payload.(runtime.ContentTyper); ok {
		if ct := t.ContentType(); ct != "" {
			return ct
		}
	}
	return fallback
}

// streamFallbackMime selects a wire content-type for a stream payload
// (io.Reader / io.ReadCloser) that has neither implemented
// `ContentType() string` nor declared an explicit value.
//
// The picker (Stage 1) ran without seeing the payload, so its choice
// may be wildly wrong for raw bytes — e.g. picking application/json
// for a payload that is just a stream of opaque data. When the
// candidate consumes list also offers application/octet-stream and
// the runtime has an octet-stream producer registered, that's a
// safer wire type than the picker's choice: it advertises "raw bytes"
// rather than making a structural claim about the body.
//
// If octet-stream is unavailable in either the candidate list or the
// producer set, the picker's choice is preserved. The wire header
// then continues to misrepresent the body — but no correct
// alternative exists and we cannot infer one without more
// information from the caller.
func streamFallbackMime(picked string, candidates []string, producers map[string]runtime.Producer) string {
	if strings.EqualFold(picked, runtime.DefaultMime) {
		return picked
	}
	for _, c := range candidates {
		if strings.EqualFold(c, runtime.DefaultMime) {
			if _, ok := producers[runtime.DefaultMime]; ok {
				return runtime.DefaultMime
			}
		}
	}
	return picked
}

func getRequestBuffer(r *request) []byte {
	if r.buf == nil {
		return nil
	}
	return r.buf.Bytes()
}

func logClose(err error, pw *io.PipeWriter) {
	log.Println(err)
	closeErr := pw.CloseWithError(err)
	if closeErr != nil {
		log.Println(closeErr)
	}
}

func mangleContentType(_, boundary string) string {
	return "multipart/form-data; boundary=" + boundary
}
