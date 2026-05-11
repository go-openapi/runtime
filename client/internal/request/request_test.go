// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package request

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	"github.com/go-openapi/runtime"
)

var testProducers = map[string]runtime.Producer{
	runtime.JSONMime: runtime.JSONProducer(),
	runtime.XMLMime:  runtime.XMLProducer(),
	runtime.TextMime: runtime.TextProducer(),
}

// Test-only constants shared across the package's *_test.go files,

const (
	valBirdWatching = "Bird watching"
	valJohn         = "John"
	valOrganTrail   = "Organ trail"
	valTom          = "Tom"
	testFile1       = "request.go"
	testFile2       = "request_test.go"
	defaultTimeout  = 10 * time.Second
)

func TestBuildRequest_SetHeaders(t *testing.T) {
	r := New(http.MethodGet, "/flats/{id}/", nil)
	_ = r.SetTimeout(defaultTimeout)

	// single value
	_ = r.SetHeaderParam("X-Rate-Limit", "500")
	assert.EqualT(t, "500", r.header.Get("X-Rate-Limit"))
	_ = r.SetHeaderParam("X-Rate-Limit", "400")
	assert.EqualT(t, "400", r.header.Get("X-Rate-Limit"))

	// multi value
	_ = r.SetHeaderParam("X-Accepts", "json", "xml", "yaml")
	assert.Equal(t, []string{"json", "xml", "yaml"}, r.header["X-Accepts"])
}

func TestBuildRequest_SetPath(t *testing.T) {
	r := New(http.MethodGet, "/flats/{id}/?hello=world", nil)

	_ = r.SetPathParam("id", "1345")
	assert.EqualT(t, "1345", r.pathParams["id"])
}

func TestBuildRequest_SetQuery(t *testing.T) {
	r := New(http.MethodGet, "/flats/{id}/", nil)

	// single value
	_ = r.SetQueryParam("hello", "there")
	assert.EqualT(t, "there", r.query.Get("hello"))

	// multi value
	_ = r.SetQueryParam("goodbye", "cruel", "world")
	assert.Equal(t, []string{"cruel", "world"}, r.query["goodbye"])
}

func TestBuildRequest_SetForm(t *testing.T) {
	// non-multipart
	r := New(http.MethodPost, "/flats", nil)
	_ = r.SetFormParam("hello", "world")
	assert.EqualT(t, "world", r.formFields.Get("hello"))
	_ = r.SetFormParam("goodbye", "cruel", "world")
	assert.Equal(t, []string{"cruel", "world"}, r.formFields["goodbye"])
}

func TestBuildRequest_SetFile(t *testing.T) {
	// needs to convert form to multipart
	r := New(http.MethodPost, "/flats/{id}/image", nil)

	// error if it isn't there
	err := r.SetFileParam("not there", os.NewFile(0, "./i-dont-exist"))
	require.Error(t, err)

	// error if it isn't a file
	err = r.SetFileParam("directory", os.NewFile(0, filepath.Join("..", "request")))
	require.Error(t, err)
	// success adds it to the map
	err = r.SetFileParam("file", mustGetFile(testFile1))
	require.NoError(t, err)
	fl, ok := r.fileFields["file"]
	require.TrueT(t, ok)
	assert.EqualT(t, testFile1, filepath.Base(fl[0].Name()))

	// success adds a file param with multiple files
	err = r.SetFileParam("otherfiles", mustGetFile(testFile1), mustGetFile(testFile2))
	require.NoError(t, err)
	fl, ok = r.fileFields["otherfiles"]
	require.TrueT(t, ok)
	assert.EqualT(t, testFile1, filepath.Base(fl[0].Name()))
	assert.EqualT(t, testFile2, filepath.Base(fl[1].Name()))
}

func mustGetFile(pth string) *os.File {
	f, err := os.Open(filepath.Join(".", pth))
	if err != nil {
		panic(err)
	}
	return f
}

func TestBuildRequest_SetBody(t *testing.T) {
	r := New(http.MethodGet, "/flats/{id}/?hello=world", nil)

	bd := []struct{ Name, Hobby string }{{valTom, valOrganTrail}, {valJohn, valBirdWatching}}

	_ = r.SetBodyParam(bd)
	assert.Equal(t, bd, r.payload)
}

func TestBuildRequest_BuildHTTPContext_PropagatesParentContext(t *testing.T) {
	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetTimeout(0) // disable per-request timeout: verify parent ctx alone flows through
		return nil
	})
	r := New(http.MethodGet, "/", reqWrtr)

	type ctxKey struct{}
	parent := context.WithValue(t.Context(), ctxKey{}, "marker")

	req, cancel, err := r.BuildHTTPContext(parent, runtime.JSONMime, "", testProducers, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, cancel)
	t.Cleanup(cancel)

	assert.EqualT(t, "marker", req.Context().Value(ctxKey{}))
	_, hasDeadline := req.Context().Deadline()
	assert.FalseT(t, hasDeadline, "no per-request timeout, no parent deadline -> request ctx must have no deadline")
}

func TestBuildRequest_BuildHTTPContext_CancelPropagates(t *testing.T) {
	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetTimeout(0)
		return nil
	})
	r := New(http.MethodGet, "/", reqWrtr)

	req, cancel, err := r.BuildHTTPContext(t.Context(), runtime.JSONMime, "", testProducers, nil, nil)
	require.NoError(t, err)
	require.NoError(t, req.Context().Err())

	cancel()
	require.ErrorIs(t, req.Context().Err(), context.Canceled)
}

func TestBuildRequest_BuildHTTPContext_AppliesPerRequestTimeout(t *testing.T) {
	const writerTimeout = 250 * time.Millisecond
	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		// ClientRequestWriter override fires inside BuildHTTP; the derived
		// ctx must observe this final value, not the runtime default.
		return req.SetTimeout(writerTimeout)
	})
	r := New(http.MethodGet, "/", reqWrtr)

	before := time.Now()
	req, cancel, err := r.BuildHTTPContext(context.Background(), runtime.JSONMime, "", testProducers, nil, nil)
	require.NoError(t, err)
	t.Cleanup(cancel)

	deadline, ok := req.Context().Deadline()
	require.TrueT(t, ok, "expected request ctx to carry a deadline from the per-request timeout")
	delta := deadline.Sub(before)
	// Loose bounds — we just want to confirm it's the writerTimeout-derived deadline,
	// not the 30s DefaultTimeout that prepareRequest seeded.
	assert.TrueT(t, delta >= writerTimeout && delta < writerTimeout+time.Second,
		"deadline should be ~%v from now, got %v", writerTimeout, delta)
}

func TestBuildRequest_BuildHTTP_NoPayload(t *testing.T) {
	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetBodyParam(nil)
		_ = req.SetQueryParam("hello", "world")
		_ = req.SetPathParam("id", "1234")
		_ = req.SetHeaderParam("X-Rate-Limit", "200")
		return nil
	})
	r := New(http.MethodPost, "/flats/{id}/", reqWrtr)

	req, cancel, err := r.BuildHTTPContext(t.Context(), runtime.JSONMime, "", testProducers, nil, nil)
	require.NoError(t, err)
	t.Cleanup(cancel)
	require.NotNil(t, req)
	assert.EqualT(t, "200", req.Header.Get(strings.ToLower("X-Rate-Limit")))
	assert.EqualT(t, "world", req.URL.Query().Get("hello"))
	assert.EqualT(t, "/flats/1234/", req.URL.Path)
}

func TestBuildRequest_BuildHTTP_Payload(t *testing.T) {
	bd := []struct{ Name, Hobby string }{{valTom, valOrganTrail}, {valJohn, valBirdWatching}}
	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetBodyParam(bd)
		_ = req.SetQueryParam("hello", "world")
		_ = req.SetPathParam("id", "1234")
		_ = req.SetHeaderParam("X-Rate-Limit", "200")
		return nil
	})
	r := New(http.MethodGet, "/flats/{id}/", reqWrtr)

	_ = r.SetHeaderParam(runtime.HeaderContentType, runtime.JSONMime)

	req, cancel, err := r.BuildHTTPContext(t.Context(), runtime.JSONMime, "", testProducers, nil, nil)
	require.NoError(t, err)
	t.Cleanup(cancel)
	require.NotNil(t, req)
	assert.EqualT(t, "200", req.Header.Get(strings.ToLower("X-Rate-Limit")))
	assert.EqualT(t, "world", req.URL.Query().Get("hello"))
	assert.EqualT(t, "/flats/1234/", req.URL.Path)
	expectedBody, err := json.Marshal(bd)
	require.NoError(t, err)
	actualBody, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	assert.Equal(t, append(expectedBody, '\n'), actualBody)
}

func TestBuildRequest_BuildHTTP_SetsInAuth(t *testing.T) {
	bd := []struct{ Name, Hobby string }{{valTom, valOrganTrail}, {valJohn, valBirdWatching}}
	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetBodyParam(bd)
		_ = req.SetQueryParam("hello", "wrong")
		_ = req.SetPathParam("id", "wrong")
		_ = req.SetHeaderParam("X-Rate-Limit", "wrong")
		return nil
	})

	auth := runtime.ClientAuthInfoWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetBodyParam(bd)
		_ = req.SetQueryParam("hello", "world")
		_ = req.SetPathParam("id", "1234")
		_ = req.SetHeaderParam("X-Rate-Limit", "200")
		return nil
	})

	r := New(http.MethodGet, "/flats/{id}/", reqWrtr)
	_ = r.SetHeaderParam(runtime.HeaderContentType, runtime.JSONMime)

	req, cancel, err := r.BuildHTTPContext(t.Context(), runtime.JSONMime, "", testProducers, nil, auth)
	require.NoError(t, err)
	t.Cleanup(cancel)
	require.NotNil(t, req)

	assert.EqualT(t, "200", req.Header.Get("X-Rate-Limit"))
	assert.EqualT(t, "world", req.URL.Query().Get("hello"))
	assert.EqualT(t, "/flats/1234/", req.URL.Path)
	expectedBody, err := json.Marshal(bd)
	require.NoError(t, err)
	actualBody, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	assert.Equal(t, append(expectedBody, '\n'), actualBody)
}

func TestBuildRequest_BuildHTTP_XMLPayload(t *testing.T) {
	bd := []struct {
		XMLName xml.Name `xml:"person"`
		Name    string   `xml:"name"`
		Hobby   string   `xml:"hobby"`
	}{{xml.Name{}, valTom, valOrganTrail}, {xml.Name{}, valJohn, valBirdWatching}}
	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetBodyParam(bd)
		_ = req.SetQueryParam("hello", "world")
		_ = req.SetPathParam("id", "1234")
		_ = req.SetHeaderParam("X-Rate-Limit", "200")
		return nil
	})
	r := New(http.MethodGet, "/flats/{id}/", reqWrtr)
	_ = r.SetHeaderParam(runtime.HeaderContentType, runtime.XMLMime)

	req, cancel, err := r.BuildHTTPContext(t.Context(), runtime.XMLMime, "", testProducers, nil, nil)
	require.NoError(t, err)
	t.Cleanup(cancel)
	require.NotNil(t, req)

	assert.EqualT(t, "200", req.Header.Get("X-Rate-Limit"))
	assert.EqualT(t, "world", req.URL.Query().Get("hello"))
	assert.EqualT(t, "/flats/1234/", req.URL.Path)
	expectedBody, err := xml.Marshal(bd)
	require.NoError(t, err)
	actualBody, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	assert.Equal(t, expectedBody, actualBody)
}

func TestBuildRequest_BuildHTTP_TextPayload(t *testing.T) {
	const bd = "Tom: Organ trail; John: Bird watching"

	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetBodyParam(bd)
		_ = req.SetQueryParam("hello", "world")
		_ = req.SetPathParam("id", "1234")
		_ = req.SetHeaderParam("X-Rate-Limit", "200")
		return nil
	})
	r := New(http.MethodGet, "/flats/{id}/", reqWrtr)
	_ = r.SetHeaderParam(runtime.HeaderContentType, runtime.TextMime)

	req, cancel, err := r.BuildHTTPContext(t.Context(), runtime.TextMime, "", testProducers, nil, nil)
	require.NoError(t, err)
	t.Cleanup(cancel)
	require.NotNil(t, req)

	assert.EqualT(t, "200", req.Header.Get("X-Rate-Limit"))
	assert.EqualT(t, "world", req.URL.Query().Get("hello"))
	assert.EqualT(t, "/flats/1234/", req.URL.Path)
	expectedBody := []byte(bd)
	actualBody, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	assert.Equal(t, expectedBody, actualBody)
}

func TestBuildRequest_BuildHTTP_Form(t *testing.T) {
	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetFormParam("something", "some value")
		_ = req.SetQueryParam("hello", "world")
		_ = req.SetPathParam("id", "1234")
		_ = req.SetHeaderParam("X-Rate-Limit", "200")
		return nil
	})
	r := New(http.MethodGet, "/flats/{id}/", reqWrtr)
	_ = r.SetHeaderParam(runtime.HeaderContentType, runtime.JSONMime)

	req, cancel, err := r.BuildHTTPContext(t.Context(), runtime.JSONMime, "", testProducers, nil, nil)
	require.NoError(t, err)
	t.Cleanup(cancel)
	require.NotNil(t, req)

	assert.EqualT(t, "200", req.Header.Get("X-Rate-Limit"))
	assert.EqualT(t, "world", req.URL.Query().Get("hello"))
	assert.EqualT(t, "/flats/1234/", req.URL.Path)
	expected := []byte("something=some+value")
	actual, _ := io.ReadAll(req.Body)
	assert.Equal(t, expected, actual)
}

func TestBuildRequest_BuildHTTP_Form_URLEncoded(t *testing.T) {
	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetFormParam("something", "some value")
		_ = req.SetQueryParam("hello", "world")
		_ = req.SetPathParam("id", "1234")
		_ = req.SetHeaderParam("X-Rate-Limit", "200")
		return nil
	})
	r := New(http.MethodGet, "/flats/{id}/", reqWrtr)
	_ = r.SetHeaderParam(runtime.HeaderContentType, runtime.URLencodedFormMime)

	req, cancel, err := r.BuildHTTPContext(t.Context(), runtime.URLencodedFormMime, "", testProducers, nil, nil)
	require.NoError(t, err)
	t.Cleanup(cancel)
	require.NotNil(t, req)

	assert.EqualT(t, "200", req.Header.Get("X-Rate-Limit"))
	assert.EqualT(t, runtime.URLencodedFormMime, req.Header.Get(runtime.HeaderContentType))
	assert.EqualT(t, "world", req.URL.Query().Get("hello"))
	assert.EqualT(t, "/flats/1234/", req.URL.Path)
	expected := []byte("something=some+value")
	actual, _ := io.ReadAll(req.Body)
	assert.Equal(t, expected, actual)
}

func TestBuildRequest_BuildHTTP_Form_Content_Length(t *testing.T) {
	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetFormParam("something", "some value")
		_ = req.SetQueryParam("hello", "world")
		_ = req.SetPathParam("id", "1234")
		_ = req.SetHeaderParam("X-Rate-Limit", "200")
		return nil
	})
	r := New(http.MethodGet, "/flats/{id}/", reqWrtr)
	_ = r.SetHeaderParam(runtime.HeaderContentType, runtime.MultipartFormMime)

	req, cancel, err := r.BuildHTTPContext(t.Context(), runtime.JSONMime, "", testProducers, nil, nil)
	require.NoError(t, err)
	t.Cleanup(cancel)
	require.NotNil(t, req)

	assert.EqualT(t, "200", req.Header.Get("X-Rate-Limit"))
	assert.EqualT(t, "world", req.URL.Query().Get("hello"))
	assert.EqualT(t, "/flats/1234/", req.URL.Path)
	assert.Condition(t, func() bool { return req.ContentLength > 0 },
		"ContentLength must great than 0. got %d", req.ContentLength)
	expected := []byte("something=some+value")
	actual, _ := io.ReadAll(req.Body)
	assert.Equal(t, expected, actual)
}

func TestBuildRequest_BuildHTTP_FormMultipart(t *testing.T) {
	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetFormParam("something", "some value")
		_ = req.SetQueryParam("hello", "world")
		_ = req.SetPathParam("id", "1234")
		_ = req.SetHeaderParam("X-Rate-Limit", "200")
		return nil
	})
	r := New(http.MethodGet, "/flats/{id}/", reqWrtr)
	_ = r.SetHeaderParam(runtime.HeaderContentType, runtime.MultipartFormMime)

	req, cancel, err := r.BuildHTTPContext(t.Context(), runtime.MultipartFormMime, "", testProducers, nil, nil)
	require.NoError(t, err)
	t.Cleanup(cancel)
	require.NotNil(t, req)

	assert.EqualT(t, "200", req.Header.Get("X-Rate-Limit"))
	assert.EqualT(t, "world", req.URL.Query().Get("hello"))
	assert.EqualT(t, "/flats/1234/", req.URL.Path)
	expected1 := []byte("Content-Disposition: form-data; name=\"something\"")
	expected2 := []byte("some value")
	actual, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	actuallines := bytes.Split(actual, []byte("\r\n"))
	assert.Len(t, actuallines, 6)
	boundary := string(actuallines[0])
	lastboundary := string(actuallines[4])
	assert.TrueT(t, strings.HasPrefix(boundary, "--"))
	assert.TrueT(t, strings.HasPrefix(lastboundary, "--") && strings.HasSuffix(lastboundary, "--"))
	assert.EqualT(t, lastboundary, boundary+"--")
	assert.Equal(t, expected1, actuallines[1])
	assert.Equal(t, expected2, actuallines[3])
}

func TestBuildRequest_BuildHTTP_FormMultiples(t *testing.T) {
	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetFormParam("something", "some value", "another value")
		_ = req.SetQueryParam("hello", "world")
		_ = req.SetPathParam("id", "1234")
		_ = req.SetHeaderParam("X-Rate-Limit", "200")
		return nil
	})
	r := New(http.MethodGet, "/flats/{id}/", reqWrtr)
	_ = r.SetHeaderParam(runtime.HeaderContentType, runtime.MultipartFormMime)

	req, cancel, err := r.BuildHTTPContext(t.Context(), runtime.MultipartFormMime, "", testProducers, nil, nil)
	require.NoError(t, err)
	t.Cleanup(cancel)
	require.NotNil(t, req)

	assert.EqualT(t, "200", req.Header.Get("X-Rate-Limit"))
	assert.EqualT(t, "world", req.URL.Query().Get("hello"))
	assert.EqualT(t, "/flats/1234/", req.URL.Path)
	expected1 := []byte("Content-Disposition: form-data; name=\"something\"")
	expected2 := []byte("some value")
	expected3 := []byte("another value")
	actual, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	actuallines := bytes.Split(actual, []byte("\r\n"))
	assert.Len(t, actuallines, 10)
	boundary := string(actuallines[0])
	lastboundary := string(actuallines[8])
	assert.TrueT(t, strings.HasPrefix(boundary, "--"))
	assert.TrueT(t, strings.HasPrefix(lastboundary, "--") && strings.HasSuffix(lastboundary, "--"))
	assert.EqualT(t, lastboundary, boundary+"--")
	assert.Equal(t, expected1, actuallines[1])
	assert.Equal(t, expected2, actuallines[3])
	assert.Equal(t, actuallines[0], actuallines[4])
	assert.Equal(t, expected1, actuallines[5])
	assert.Equal(t, expected3, actuallines[7])
}

func TestBuildRequest_BuildHTTP_Files(t *testing.T) {
	tmpDir := t.TempDir()
	cont, err := os.ReadFile(testFile1)
	require.NoError(t, err)
	cont2, err := os.ReadFile(testFile2)
	require.NoError(t, err)
	emptyFile, err := os.CreateTemp(tmpDir, "empty")
	require.NoError(t, err)

	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetFormParam("something", "some value")
		_ = req.SetFileParam("file", mustGetFile(testFile1))
		_ = req.SetFileParam("otherfiles", mustGetFile(testFile1), mustGetFile(testFile2))
		_ = req.SetFileParam("empty", emptyFile)
		_ = req.SetQueryParam("hello", "world")
		_ = req.SetPathParam("id", "1234")
		_ = req.SetHeaderParam("X-Rate-Limit", "200")
		return nil
	})
	r := New(http.MethodGet, "/flats/{id}/", reqWrtr)
	_ = r.SetHeaderParam(runtime.HeaderContentType, runtime.JSONMime)
	req, cancel, err := r.BuildHTTPContext(t.Context(), runtime.JSONMime, "", testProducers, nil, nil)
	require.NoError(t, err)
	t.Cleanup(cancel)
	require.NotNil(t, req)

	assert.EqualT(t, "200", req.Header.Get("X-Rate-Limit"))
	assert.EqualT(t, "world", req.URL.Query().Get("hello"))
	assert.EqualT(t, "/flats/1234/", req.URL.Path)

	mediaType, params, err := mime.ParseMediaType(req.Header.Get(runtime.HeaderContentType))
	require.NoError(t, err)

	assert.EqualT(t, runtime.MultipartFormMime, mediaType)
	boundary := params["boundary"]
	mr := multipart.NewReader(req.Body, boundary)
	defer req.Body.Close()
	frm, err := mr.ReadForm(1 << 20)
	require.NoError(t, err)

	assert.EqualT(t, "some value", frm.Value["something"][0])
	fileverifier := func(name string, index int, filename string, content []byte) {
		mpff := frm.File[name][index]
		mpf, e := mpff.Open()
		require.NoError(t, e)
		defer mpf.Close()
		assert.EqualT(t, filename, mpff.Filename)
		actual, e := io.ReadAll(mpf)
		require.NoError(t, e)
		assert.Equal(t, content, actual)
	}
	fileverifier("file", 0, testFile1, cont)
	fileverifier("otherfiles", 0, testFile1, cont)
	fileverifier("otherfiles", 1, testFile2, cont2)
	fileverifier("empty", 0, filepath.Base(emptyFile.Name()), []byte{})
}

// TestBuildRequest_BuildHTTP_Files_URLEncoded covers issue #286: when the
// caller explicitly picks application/x-www-form-urlencoded, file fields must
// be encoded as regular URL-encoded form values rather than producing a
// multipart body advertised under a urlencoded Content-Type.
func TestBuildRequest_BuildHTTP_Files_URLEncoded(t *testing.T) {
	cont, err := os.ReadFile(testFile2)
	require.NoError(t, err)
	cont2, err := os.ReadFile(testFile1)
	require.NoError(t, err)

	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetFormParam("something", "some value")
		_ = req.SetFileParam("file", mustGetFile(testFile2))
		_ = req.SetFileParam("otherfiles", mustGetFile(testFile2), mustGetFile(testFile1))
		_ = req.SetQueryParam("hello", "world")
		_ = req.SetPathParam("id", "1234")
		_ = req.SetHeaderParam("X-Rate-Limit", "200")
		return nil
	})
	r := New(http.MethodGet, "/flats/{id}/", reqWrtr)
	_ = r.SetHeaderParam(runtime.HeaderContentType, runtime.URLencodedFormMime)
	req, cancel, err := r.BuildHTTPContext(t.Context(), runtime.URLencodedFormMime, "", testProducers, nil, nil)
	require.NoError(t, err)
	t.Cleanup(cancel)
	require.NotNil(t, req)

	assert.EqualT(t, "200", req.Header.Get("X-Rate-Limit"))
	assert.EqualT(t, "world", req.URL.Query().Get("hello"))
	assert.EqualT(t, "/flats/1234/", req.URL.Path)

	// Content-Type must be the bare urlencoded type — no boundary parameter.
	assert.EqualT(t, runtime.URLencodedFormMime, req.Header.Get(runtime.HeaderContentType))

	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	defer req.Body.Close()

	values, err := url.ParseQuery(string(body))
	require.NoError(t, err)

	assert.EqualT(t, "some value", values.Get("something"))
	require.Len(t, values["file"], 1)
	assert.Equal(t, string(cont), values["file"][0])
	require.Len(t, values["otherfiles"], 2)
	assert.Equal(t, string(cont), values["otherfiles"][0])
	assert.Equal(t, string(cont2), values["otherfiles"][1])
}

type contentTypeProvider struct {
	runtime.NamedReadCloser

	contentType string
}

func (p contentTypeProvider) ContentType() string {
	return p.contentType
}

func TestBuildRequest_BuildHTTP_File_ContentType(t *testing.T) {
	cont, err := os.ReadFile(testFile1)
	require.NoError(t, err)
	cont2, err := os.ReadFile(testFile2)
	require.NoError(t, err)

	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetPathParam("id", "1234")
		_ = req.SetFileParam("file1", contentTypeProvider{
			NamedReadCloser: mustGetFile(testFile1),
			contentType:     runtime.DefaultMime,
		})
		_ = req.SetFileParam("file2", mustGetFile(testFile2))

		return nil
	})
	r := New(http.MethodGet, "/flats/{id}/", reqWrtr)
	_ = r.SetHeaderParam(runtime.HeaderContentType, runtime.JSONMime)
	req, cancel, err := r.BuildHTTPContext(t.Context(), runtime.JSONMime, "", testProducers, nil, nil)
	require.NoError(t, err)
	t.Cleanup(cancel)
	require.NotNil(t, req)

	assert.EqualT(t, "/flats/1234/", req.URL.Path)
	mediaType, params, err := mime.ParseMediaType(req.Header.Get(runtime.HeaderContentType))
	require.NoError(t, err)
	assert.EqualT(t, runtime.MultipartFormMime, mediaType)
	boundary := params["boundary"]
	mr := multipart.NewReader(req.Body, boundary)
	defer req.Body.Close()
	frm, err := mr.ReadForm(1 << 20)
	require.NoError(t, err)

	fileverifier := func(name string, index int, filename string, content []byte, contentType string) {
		mpff := frm.File[name][index]
		mpf, e := mpff.Open()
		require.NoError(t, e)
		defer mpf.Close()
		assert.EqualT(t, filename, mpff.Filename)
		actual, e := io.ReadAll(mpf)
		require.NoError(t, e)
		assert.Equal(t, content, actual)
		assert.EqualT(t, mpff.Header.Get("Content-Type"), contentType)
	}
	fileverifier("file1", 0, testFile1, cont, runtime.DefaultMime)
	fileverifier("file2", 0, testFile2, cont2, "text/plain; charset=utf-8")
}

func TestBuildRequest_BuildHTTP_BasePath(t *testing.T) {
	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetBodyParam(nil)
		_ = req.SetQueryParam("hello", "world")
		_ = req.SetPathParam("id", "1234")
		_ = req.SetHeaderParam("X-Rate-Limit", "200")
		return nil
	})
	r := New(http.MethodPost, "/flats/{id}/", reqWrtr)

	req, cancel, err := r.BuildHTTPContext(t.Context(), runtime.JSONMime, "/basepath", testProducers, nil, nil)
	require.NoError(t, err)
	t.Cleanup(cancel)
	require.NotNil(t, req)
	assert.EqualT(t, "200", req.Header.Get("X-Rate-Limit"))
	assert.EqualT(t, "world", req.URL.Query().Get("hello"))
	assert.EqualT(t, "/basepath/flats/1234/", req.URL.Path)
}

func TestBuildRequest_BuildHTTP_EscapedPath(t *testing.T) {
	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetBodyParam(nil)
		_ = req.SetQueryParam("hello", "world")
		_ = req.SetPathParam("id", "1234/?*&^%")
		_ = req.SetHeaderParam("X-Rate-Limit", "200")
		return nil
	})
	r := New(http.MethodPost, "/flats/{id}/", reqWrtr)

	req, cancel, err := r.BuildHTTPContext(t.Context(), runtime.JSONMime, "/basepath", testProducers, nil, nil)
	require.NoError(t, err)
	t.Cleanup(cancel)
	require.NotNil(t, req)

	assert.EqualT(t, "200", req.Header.Get("X-Rate-Limit"))
	assert.EqualT(t, "world", req.URL.Query().Get("hello"))
	assert.EqualT(t, "/basepath/flats/1234/?*&^%/", req.URL.Path)
	assert.EqualT(t, "/basepath/flats/1234%2F%3F%2A&%5E%25/", req.URL.RawPath)
	assert.EqualT(t, req.URL.RawPath, req.URL.EscapedPath())
}

// TestBuildRequest_BuildHTTP_RootPathTrailingSlash locks in the fix for
// issue #101: the bare-root pattern "/" under a non-empty basePath must
// keep its trailing slash, and the bare-root cases that the pre-fix
// formula avoided ("" / "/" basePath) must still not produce "//".
func TestBuildRequest_BuildHTTP_RootPathTrailingSlash(t *testing.T) {
	const bp = "/basepath"
	cases := []struct {
		name        string
		basePath    string
		pathPattern string
		wantPath    string
	}{
		{"root pattern under non-empty basePath keeps slash (#101)", bp, "/", bp + "/"},
		{"root pattern under '/' basePath stays '/'", "/", "/", "/"},
		{"root pattern under empty basePath stays '/'", "", "/", "/"},
		{"non-root trailing slash still preserved", bp, "/users/", bp + "/users/"},
		{"no trailing slash on pattern produces no trailing slash", bp, "/users", bp + "/users"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reqWrtr := runtime.ClientRequestWriterFunc(func(_ runtime.ClientRequest, _ strfmt.Registry) error {
				return nil
			})
			r := New(http.MethodGet, tc.pathPattern, reqWrtr)

			req, cancel, err := r.BuildHTTPContext(t.Context(), runtime.JSONMime, tc.basePath, testProducers, nil, nil)
			require.NoError(t, err)
			t.Cleanup(cancel)
			require.NotNil(t, req)

			assert.EqualT(t, tc.wantPath, req.URL.Path)
		})
	}
}

func TestBuildRequest_BuildHTTP_BasePathWithQueryParameters(t *testing.T) {
	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetBodyParam(nil)
		_ = req.SetQueryParam("hello", "world")
		_ = req.SetPathParam("id", "1234")
		return nil
	})
	r := New(http.MethodPost, "/flats/{id}/", reqWrtr)

	req, cancel, err := r.BuildHTTPContext(t.Context(), runtime.JSONMime, "/basepath?foo=bar", testProducers, nil, nil)
	require.NoError(t, err)
	t.Cleanup(cancel)
	require.NotNil(t, req)

	assert.EqualT(t, "world", req.URL.Query().Get("hello"))
	assert.EqualT(t, "bar", req.URL.Query().Get("foo"))
	assert.EqualT(t, "/basepath/flats/1234/", req.URL.Path)
}

func TestBuildRequest_BuildHTTP_PathPatternWithQueryParameters(t *testing.T) {
	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetBodyParam(nil)
		_ = req.SetQueryParam("hello", "world")
		_ = req.SetPathParam("id", "1234")
		return nil
	})
	r := New(http.MethodPost, "/flats/{id}/?foo=bar", reqWrtr)

	req, cancel, err := r.BuildHTTPContext(t.Context(), runtime.JSONMime, "/basepath", testProducers, nil, nil)
	require.NoError(t, err)
	t.Cleanup(cancel)
	require.NotNil(t, req)

	assert.EqualT(t, "world", req.URL.Query().Get("hello"))
	assert.EqualT(t, "bar", req.URL.Query().Get("foo"))
	assert.EqualT(t, "/basepath/flats/1234/", req.URL.Path)
}

func TestBuildRequest_BuildHTTP_StaticParametersPathPatternPrevails(t *testing.T) {
	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetBodyParam(nil)
		_ = req.SetPathParam("id", "1234")
		return nil
	})
	r := New(http.MethodPost, "/flats/{id}/?hello=world", reqWrtr)

	req, cancel, err := r.BuildHTTPContext(t.Context(), runtime.JSONMime, "/basepath?hello=kitty", testProducers, nil, nil)
	require.NoError(t, err)
	t.Cleanup(cancel)
	require.NotNil(t, req)

	assert.EqualT(t, "world", req.URL.Query().Get("hello"))
	assert.EqualT(t, "/basepath/flats/1234/", req.URL.Path)
}

func TestBuildRequest_BuildHTTP_StaticParametersConflictClientPrevails(t *testing.T) {
	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetBodyParam(nil)
		_ = req.SetQueryParam("hello", "there")
		_ = req.SetPathParam("id", "1234")
		return nil
	})
	r := New(http.MethodPost, "/flats/{id}/?hello=world", reqWrtr)

	req, cancel, err := r.BuildHTTPContext(t.Context(), runtime.JSONMime, "/basepath?hello=kitty", testProducers, nil, nil)
	require.NoError(t, err)
	t.Cleanup(cancel)
	require.NotNil(t, req)

	assert.EqualT(t, "there", req.URL.Query().Get("hello"))
	assert.EqualT(t, "/basepath/flats/1234/", req.URL.Path)
}

// TestBuildRequest_BuildHTTP_ParametrizedFormMimes verifies that
// isMultipart correctly handles legal RFC 7231 parameters and case
// variants on the form mime types — i.e. it strips `; boundary=…`,
// `; charset=…`, etc. before comparing against the canonical constants.
//
// Without the fix, three things go wrong:
//   - `multipart/form-data; boundary=xyz` with no files routes to the
//     buffered/urlencoded flow, producing a urlencoded body with a
//     multipart Content-Type header.
//   - `application/x-www-form-urlencoded; charset=utf-8` with file
//     fields silently switches to multipart, undoing the urlencoded
//     short-circuit that #286 added.
//   - Mixed-case `Multipart/Form-Data` misses the multipart compare
//     (it's exact, not case-insensitive).
func TestBuildRequest_BuildHTTP_ParametrizedFormMimes(t *testing.T) {
	t.Run("multipart with boundary param, no files", func(t *testing.T) {
		reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
			return req.SetFormParam("name", "fido")
		})
		r := New(http.MethodPost, "/", reqWrtr)

		// A caller-supplied boundary that the runtime would normally
		// add itself — the dispatch must still recognize the base mime
		// as multipart and route to the streaming flow.
		mt := runtime.MultipartFormMime + "; boundary=caller-supplied"
		req, cancel, err := r.BuildHTTPContext(t.Context(), mt, "", testProducers, nil, nil)
		require.NoError(t, err)
		t.Cleanup(cancel)
		require.NotNil(t, req)

		// Streaming flow taken: req.Body is a pipe reader, and the
		// runtime emitted its own boundary (mangleContentType
		// overwrites the caller's parameter).
		ct := req.Header.Get(runtime.HeaderContentType)
		assert.TrueT(t, strings.HasPrefix(ct, runtime.MultipartFormMime+"; boundary="),
			"expected multipart Content-Type with runtime boundary, got %q", ct)

		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), `name="name"`)
		assert.Contains(t, string(body), "fido")
	})

	t.Run("urlencoded with charset param, with files", func(t *testing.T) {
		reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
			return req.SetFileParam("upload", runtime.NamedReader("doc.txt", strings.NewReader("abc")))
		})
		r := New(http.MethodPost, "/", reqWrtr)

		// Per #286, urlencoded-with-files is honored: the file content
		// travels inline as a regular form value. The charset param
		// must not break the short-circuit.
		mt := runtime.URLencodedFormMime + "; charset=utf-8"
		req, cancel, err := r.BuildHTTPContext(t.Context(), mt, "", testProducers, nil, nil)
		require.NoError(t, err)
		t.Cleanup(cancel)
		require.NotNil(t, req)

		// Buffered/urlencoded flow taken: CT is set verbatim to the
		// caller's mediaType (params preserved); body is the inlined
		// form value.
		assert.EqualT(t, mt, req.Header.Get(runtime.HeaderContentType))
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		assert.EqualT(t, "upload=abc", string(body))
	})

	t.Run("multipart in mixed case", func(t *testing.T) {
		reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
			return req.SetFormParam("name", "fido")
		})
		r := New(http.MethodPost, "/", reqWrtr)

		req, cancel, err := r.BuildHTTPContext(t.Context(), "Multipart/Form-Data", "", testProducers, nil, nil)
		require.NoError(t, err)
		t.Cleanup(cancel)
		require.NotNil(t, req)

		// Case-insensitive recognition: streaming flow taken.
		ct := req.Header.Get(runtime.HeaderContentType)
		assert.TrueT(t, strings.HasPrefix(ct, runtime.MultipartFormMime+"; boundary="),
			"expected multipart Content-Type with boundary, got %q", ct)
	})
}

// TestBuildRequest_BuildHTTP_EmptyForm verifies that a request with
// neither form fields, file fields, nor payload routes through the
// buffered flow regardless of mediaType — and produces a well-formed
// no-body request. In particular, the multipart mime case must NOT
// engage the streaming flow when there is nothing to stream (which
// would spawn an idle goroutine and produce a pipe-backed body).
func TestBuildRequest_BuildHTTP_EmptyForm(t *testing.T) {
	cases := []struct {
		name      string
		mediaType string
	}{
		{"empty + multipart mime", runtime.MultipartFormMime},
		{"empty + urlencoded mime", runtime.URLencodedFormMime},
		{"empty + json mime", runtime.JSONMime},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
				_ = req.SetQueryParam("hello", "world")
				_ = req.SetPathParam("id", "1234")
				return nil
			})
			r := New(http.MethodPost, "/flats/{id}/", reqWrtr)

			req, cancel, err := r.BuildHTTPContext(t.Context(), tc.mediaType, "", testProducers, nil, nil)
			require.NoError(t, err)
			t.Cleanup(cancel)
			require.NotNil(t, req)

			// No body source: confirms the streaming flow was not taken
			// (otherwise req.Body would be a pipe reader).
			assert.Nil(t, req.Body)

			// No Content-Type: there is no body to describe and the
			// trailing fallback only fires when body != nil.
			assert.Empty(t, req.Header.Get(runtime.HeaderContentType))

			// URL/query wiring still applies.
			assert.EqualT(t, "/flats/1234/", req.URL.Path)
			assert.EqualT(t, "world", req.URL.Query().Get("hello"))
		})
	}
}

// observableFile is a NamedReadCloser that signals on Close and never
// blocks on Read. Used to verify that error paths in buildHTTP close
// the body source — and, for multipart, that the spawned writer
// goroutine terminates and runs its deferred file-close loop.
type observableFile struct {
	name   string
	data   *bytes.Reader
	closed chan struct{}
}

func newObservableFile(name string, data []byte) *observableFile {
	return &observableFile{
		name:   name,
		data:   bytes.NewReader(data),
		closed: make(chan struct{}),
	}
}

func (f *observableFile) Read(p []byte) (int, error) { return f.data.Read(p) }
func (f *observableFile) Name() string               { return f.name }
func (f *observableFile) Close() error {
	select {
	case <-f.closed:
		// already closed
	default:
		close(f.closed)
	}
	return nil
}

// TestBuildRequest_BuildHTTP_MultipartGoroutineCleanupOnAuthError is a
// regression test for a goroutine leak: when auth fails after
// writeMultipartBody has spawned the pipe-writer goroutine, the
// goroutine would park forever on pw.Write because no consumer ever
// reads the pipe. The fix in buildStreamingRequest closes the pipe
// reader on error paths, which unblocks the writer goroutine and lets
// it run its deferred file-close.
func TestBuildRequest_BuildHTTP_MultipartGoroutineCleanupOnAuthError(t *testing.T) {
	file := newObservableFile("data.bin", bytes.Repeat([]byte("x"), 4096))

	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		return req.SetFileParam("upload", file)
	})

	authErr := errors.New("auth failed")
	auth := runtime.ClientAuthInfoWriterFunc(func(_ runtime.ClientRequest, _ strfmt.Registry) error {
		return authErr
	})

	r := New(http.MethodPost, "/upload", reqWrtr)
	req, cancel, err := r.BuildHTTPContext(t.Context(), runtime.MultipartFormMime, "", testProducers, nil, auth)
	require.ErrorIs(t, err, authErr)
	t.Cleanup(cancel)
	require.Nil(t, req)

	// The multipart goroutine must terminate (its deferred file-close
	// runs and signals on f.closed). Without the fix this select would
	// hit the timeout because the goroutine is parked on pw.Write.
	select {
	case <-file.closed:
	case <-time.After(2 * time.Second):
		t.Fatal("multipart goroutine leaked: file was never closed after auth error")
	}
}

// observableReadCloser is a stream payload whose Close is observable.
type observableReadCloser struct {
	data   *bytes.Reader
	closed chan struct{}
}

func newObservableReadCloser(data []byte) *observableReadCloser {
	return &observableReadCloser{data: bytes.NewReader(data), closed: make(chan struct{})}
}

func (r *observableReadCloser) Read(p []byte) (int, error) { return r.data.Read(p) }
func (r *observableReadCloser) Close() error {
	select {
	case <-r.closed:
	default:
		close(r.closed)
	}
	return nil
}

// TestBuildRequest_BuildHTTP_StreamPayloadClosedOnAuthError verifies
// that a stream payload's io.ReadCloser is closed when auth fails —
// otherwise the user-provided closer leaks.
func TestBuildRequest_BuildHTTP_StreamPayloadClosedOnAuthError(t *testing.T) {
	payload := newObservableReadCloser([]byte("hello"))

	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		return req.SetBodyParam(payload)
	})

	authErr := errors.New("auth failed")
	auth := runtime.ClientAuthInfoWriterFunc(func(_ runtime.ClientRequest, _ strfmt.Registry) error {
		return authErr
	})

	r := New(http.MethodPost, "/stream", reqWrtr)
	req, cancel, err := r.BuildHTTPContext(t.Context(), runtime.JSONMime, "", testProducers, nil, auth)
	require.ErrorIs(t, err, authErr)
	t.Cleanup(cancel)
	require.Nil(t, req)

	select {
	case <-payload.closed:
	case <-time.After(2 * time.Second):
		t.Fatal("stream payload leaked: ReadCloser was never closed after auth error")
	}
}

// TestBuildRequest_BuildHTTPContext_MultipartCancelAbortsUpload verifies that
// canceling the parent context aborts an in-flight multipart upload: the
// pipe consumer surfaces context.Canceled and the spawned writer goroutine
// terminates, running its deferred file-close.
//
// Without ctx wiring inside streamMultipartParts, cancellation would only
// take effect once the http transport noticed and closed the body reader,
// which is too late for an upload that has not yet been handed to a
// transport (e.g. test code reading req.Body directly).
func TestBuildRequest_BuildHTTPContext_MultipartCancelAbortsUpload(t *testing.T) {
	// 4 MiB ensures io.Copy iterates over multiple buffer-sized Reads,
	// so the ctxReader gets several chances to observe cancellation.
	file := newObservableFile("big.bin", bytes.Repeat([]byte("x"), 4<<20))

	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		return req.SetFileParam("upload", file)
	})

	r := New(http.MethodPost, "/upload", reqWrtr)

	parentCtx, parentCancel := context.WithCancel(context.Background())

	req, cancel, err := r.BuildHTTPContext(parentCtx, runtime.MultipartFormMime, "", testProducers, nil, nil)
	require.NoError(t, err)
	t.Cleanup(cancel)

	// Cancel before draining the body. The streaming goroutine may already
	// have parked on a pw.Write inside the part header; once we begin
	// reading, the next ctxReader.Read sees the canceled ctx and the pipe
	// is closed with context.Canceled.
	parentCancel()

	_, err = io.ReadAll(req.Body)
	require.ErrorIs(t, err, context.Canceled)

	select {
	case <-file.closed:
	case <-time.After(2 * time.Second):
		t.Fatal("multipart goroutine did not close file after cancellation")
	}
}
