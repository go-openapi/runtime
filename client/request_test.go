// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-openapi/runtime"
)

var testProducers = map[string]runtime.Producer{
	runtime.JSONMime: runtime.JSONProducer(),
	runtime.XMLMime:  runtime.XMLProducer(),
	runtime.TextMime: runtime.TextProducer(),
}

func TestBuildRequest_SetHeaders(t *testing.T) {
	r := newRequest(http.MethodGet, "/flats/{id}/", nil)

	// single value
	_ = r.SetHeaderParam("X-Rate-Limit", "500")
	assert.Equal(t, "500", r.header.Get("X-Rate-Limit"))
	_ = r.SetHeaderParam("X-Rate-Limit", "400")
	assert.Equal(t, "400", r.header.Get("X-Rate-Limit"))

	// multi value
	_ = r.SetHeaderParam("X-Accepts", "json", "xml", "yaml")
	assert.EqualValues(t, []string{"json", "xml", "yaml"}, r.header["X-Accepts"])
}

func TestBuildRequest_SetPath(t *testing.T) {
	r := newRequest(http.MethodGet, "/flats/{id}/?hello=world", nil)

	_ = r.SetPathParam("id", "1345")
	assert.Equal(t, "1345", r.pathParams["id"])
}

func TestBuildRequest_SetQuery(t *testing.T) {
	r := newRequest(http.MethodGet, "/flats/{id}/", nil)

	// single value
	_ = r.SetQueryParam("hello", "there")
	assert.Equal(t, "there", r.query.Get("hello"))

	// multi value
	_ = r.SetQueryParam("goodbye", "cruel", "world")
	assert.Equal(t, []string{"cruel", "world"}, r.query["goodbye"])
}

func TestBuildRequest_SetForm(t *testing.T) {
	// non-multipart
	r := newRequest(http.MethodPost, "/flats", nil)
	_ = r.SetFormParam("hello", "world")
	assert.Equal(t, "world", r.formFields.Get("hello"))
	_ = r.SetFormParam("goodbye", "cruel", "world")
	assert.Equal(t, []string{"cruel", "world"}, r.formFields["goodbye"])
}

func TestBuildRequest_SetFile(t *testing.T) {
	// needs to convert form to multipart
	r := newRequest(http.MethodPost, "/flats/{id}/image", nil)

	// error if it isn't there
	err := r.SetFileParam("not there", os.NewFile(0, "./i-dont-exist"))
	require.Error(t, err)

	// error if it isn't a file
	err = r.SetFileParam("directory", os.NewFile(0, "../client"))
	require.Error(t, err)
	// success adds it to the map
	err = r.SetFileParam("file", mustGetFile("./runtime.go"))
	require.NoError(t, err)
	fl, ok := r.fileFields["file"]
	require.True(t, ok)
	assert.Equal(t, "runtime.go", filepath.Base(fl[0].Name()))

	// success adds a file param with multiple files
	err = r.SetFileParam("otherfiles", mustGetFile("./runtime.go"), mustGetFile("./request.go"))
	require.NoError(t, err)
	fl, ok = r.fileFields["otherfiles"]
	require.True(t, ok)
	assert.Equal(t, "runtime.go", filepath.Base(fl[0].Name()))
	assert.Equal(t, "request.go", filepath.Base(fl[1].Name()))
}

func mustGetFile(path string) *os.File {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	return f
}

func TestBuildRequest_SetBody(t *testing.T) {
	r := newRequest(http.MethodGet, "/flats/{id}/?hello=world", nil)

	bd := []struct{ Name, Hobby string }{{"Tom", "Organ trail"}, {"John", "Bird watching"}}

	_ = r.SetBodyParam(bd)
	assert.Equal(t, bd, r.payload)
}

func TestBuildRequest_BuildHTTP_NoPayload(t *testing.T) {
	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetBodyParam(nil)
		_ = req.SetQueryParam("hello", "world")
		_ = req.SetPathParam("id", "1234")
		_ = req.SetHeaderParam("X-Rate-Limit", "200")
		return nil
	})
	r := newRequest(http.MethodPost, "/flats/{id}/", reqWrtr)

	req, err := r.BuildHTTP(runtime.JSONMime, "", testProducers, nil)
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, "200", req.Header.Get("x-rate-limit"))
	assert.Equal(t, "world", req.URL.Query().Get("hello"))
	assert.Equal(t, "/flats/1234/", req.URL.Path)
}

func TestBuildRequest_BuildHTTP_Payload(t *testing.T) {
	bd := []struct{ Name, Hobby string }{{"Tom", "Organ trail"}, {"John", "Bird watching"}}
	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetBodyParam(bd)
		_ = req.SetQueryParam("hello", "world")
		_ = req.SetPathParam("id", "1234")
		_ = req.SetHeaderParam("X-Rate-Limit", "200")
		return nil
	})
	r := newRequest(http.MethodGet, "/flats/{id}/", reqWrtr)
	_ = r.SetHeaderParam(runtime.HeaderContentType, runtime.JSONMime)

	req, err := r.BuildHTTP(runtime.JSONMime, "", testProducers, nil)
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, "200", req.Header.Get("x-rate-limit"))
	assert.Equal(t, "world", req.URL.Query().Get("hello"))
	assert.Equal(t, "/flats/1234/", req.URL.Path)
	expectedBody, err := json.Marshal(bd)
	require.NoError(t, err)
	actualBody, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	assert.Equal(t, append(expectedBody, '\n'), actualBody)
}

func TestBuildRequest_BuildHTTP_SetsInAuth(t *testing.T) {
	bd := []struct{ Name, Hobby string }{{"Tom", "Organ trail"}, {"John", "Bird watching"}}
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

	r := newRequest(http.MethodGet, "/flats/{id}/", reqWrtr)
	_ = r.SetHeaderParam(runtime.HeaderContentType, runtime.JSONMime)

	req, err := r.buildHTTP(runtime.JSONMime, "", testProducers, nil, auth)
	require.NoError(t, err)
	require.NotNil(t, req)

	assert.Equal(t, "200", req.Header.Get("x-rate-limit"))
	assert.Equal(t, "world", req.URL.Query().Get("hello"))
	assert.Equal(t, "/flats/1234/", req.URL.Path)
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
	}{{xml.Name{}, "Tom", "Organ trail"}, {xml.Name{}, "John", "Bird watching"}}
	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetBodyParam(bd)
		_ = req.SetQueryParam("hello", "world")
		_ = req.SetPathParam("id", "1234")
		_ = req.SetHeaderParam("X-Rate-Limit", "200")
		return nil
	})
	r := newRequest(http.MethodGet, "/flats/{id}/", reqWrtr)
	_ = r.SetHeaderParam(runtime.HeaderContentType, runtime.XMLMime)

	req, err := r.BuildHTTP(runtime.XMLMime, "", testProducers, nil)
	require.NoError(t, err)
	require.NotNil(t, req)

	assert.Equal(t, "200", req.Header.Get("x-rate-limit"))
	assert.Equal(t, "world", req.URL.Query().Get("hello"))
	assert.Equal(t, "/flats/1234/", req.URL.Path)
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
	r := newRequest(http.MethodGet, "/flats/{id}/", reqWrtr)
	_ = r.SetHeaderParam(runtime.HeaderContentType, runtime.TextMime)

	req, err := r.BuildHTTP(runtime.TextMime, "", testProducers, nil)
	require.NoError(t, err)
	require.NotNil(t, req)

	assert.Equal(t, "200", req.Header.Get("x-rate-limit"))
	assert.Equal(t, "world", req.URL.Query().Get("hello"))
	assert.Equal(t, "/flats/1234/", req.URL.Path)
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
	r := newRequest(http.MethodGet, "/flats/{id}/", reqWrtr)
	_ = r.SetHeaderParam(runtime.HeaderContentType, runtime.JSONMime)

	req, err := r.BuildHTTP(runtime.JSONMime, "", testProducers, nil)
	require.NoError(t, err)
	require.NotNil(t, req)

	assert.Equal(t, "200", req.Header.Get("x-rate-limit"))
	assert.Equal(t, "world", req.URL.Query().Get("hello"))
	assert.Equal(t, "/flats/1234/", req.URL.Path)
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
	r := newRequest(http.MethodGet, "/flats/{id}/", reqWrtr)
	_ = r.SetHeaderParam(runtime.HeaderContentType, runtime.URLencodedFormMime)

	req, err := r.BuildHTTP(runtime.URLencodedFormMime, "", testProducers, nil)
	require.NoError(t, err)
	require.NotNil(t, req)

	assert.Equal(t, "200", req.Header.Get("x-rate-limit"))
	assert.Equal(t, runtime.URLencodedFormMime, req.Header.Get(runtime.HeaderContentType))
	assert.Equal(t, "world", req.URL.Query().Get("hello"))
	assert.Equal(t, "/flats/1234/", req.URL.Path)
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
	r := newRequest(http.MethodGet, "/flats/{id}/", reqWrtr)
	_ = r.SetHeaderParam(runtime.HeaderContentType, runtime.MultipartFormMime)

	req, err := r.BuildHTTP(runtime.JSONMime, "", testProducers, nil)
	require.NoError(t, err)
	require.NotNil(t, req)

	assert.Equal(t, "200", req.Header.Get("x-rate-limit"))
	assert.Equal(t, "world", req.URL.Query().Get("hello"))
	assert.Equal(t, "/flats/1234/", req.URL.Path)
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
	r := newRequest(http.MethodGet, "/flats/{id}/", reqWrtr)
	_ = r.SetHeaderParam(runtime.HeaderContentType, runtime.MultipartFormMime)

	req, err := r.BuildHTTP(runtime.MultipartFormMime, "", testProducers, nil)
	require.NoError(t, err)
	require.NotNil(t, req)

	assert.Equal(t, "200", req.Header.Get("x-rate-limit"))
	assert.Equal(t, "world", req.URL.Query().Get("hello"))
	assert.Equal(t, "/flats/1234/", req.URL.Path)
	expected1 := []byte("Content-Disposition: form-data; name=\"something\"")
	expected2 := []byte("some value")
	actual, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	actuallines := bytes.Split(actual, []byte("\r\n"))
	assert.Len(t, actuallines, 6)
	boundary := string(actuallines[0])
	lastboundary := string(actuallines[4])
	assert.True(t, strings.HasPrefix(boundary, "--"))
	assert.True(t, strings.HasPrefix(lastboundary, "--") && strings.HasSuffix(lastboundary, "--"))
	assert.Equal(t, lastboundary, boundary+"--")
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
	r := newRequest(http.MethodGet, "/flats/{id}/", reqWrtr)
	_ = r.SetHeaderParam(runtime.HeaderContentType, runtime.MultipartFormMime)

	req, err := r.BuildHTTP(runtime.MultipartFormMime, "", testProducers, nil)
	require.NoError(t, err)
	require.NotNil(t, req)

	assert.Equal(t, "200", req.Header.Get("x-rate-limit"))
	assert.Equal(t, "world", req.URL.Query().Get("hello"))
	assert.Equal(t, "/flats/1234/", req.URL.Path)
	expected1 := []byte("Content-Disposition: form-data; name=\"something\"")
	expected2 := []byte("some value")
	expected3 := []byte("another value")
	actual, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	actuallines := bytes.Split(actual, []byte("\r\n"))
	assert.Len(t, actuallines, 10)
	boundary := string(actuallines[0])
	lastboundary := string(actuallines[8])
	assert.True(t, strings.HasPrefix(boundary, "--"))
	assert.True(t, strings.HasPrefix(lastboundary, "--") && strings.HasSuffix(lastboundary, "--"))
	assert.Equal(t, lastboundary, boundary+"--")
	assert.Equal(t, expected1, actuallines[1])
	assert.Equal(t, expected2, actuallines[3])
	assert.Equal(t, actuallines[0], actuallines[4])
	assert.Equal(t, expected1, actuallines[5])
	assert.Equal(t, expected3, actuallines[7])
}

func TestBuildRequest_BuildHTTP_Files(t *testing.T) {
	cont, err := os.ReadFile("./runtime.go")
	require.NoError(t, err)
	cont2, err := os.ReadFile("./request.go")
	require.NoError(t, err)
	emptyFile, err := os.CreateTemp("", "empty")
	require.NoError(t, err)

	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetFormParam("something", "some value")
		_ = req.SetFileParam("file", mustGetFile("./runtime.go"))
		_ = req.SetFileParam("otherfiles", mustGetFile("./runtime.go"), mustGetFile("./request.go"))
		_ = req.SetFileParam("empty", emptyFile)
		_ = req.SetQueryParam("hello", "world")
		_ = req.SetPathParam("id", "1234")
		_ = req.SetHeaderParam("X-Rate-Limit", "200")
		return nil
	})
	r := newRequest(http.MethodGet, "/flats/{id}/", reqWrtr)
	_ = r.SetHeaderParam(runtime.HeaderContentType, runtime.JSONMime)
	req, err := r.BuildHTTP(runtime.JSONMime, "", testProducers, nil)
	require.NoError(t, err)
	require.NotNil(t, req)

	assert.Equal(t, "200", req.Header.Get("x-rate-limit"))
	assert.Equal(t, "world", req.URL.Query().Get("hello"))
	assert.Equal(t, "/flats/1234/", req.URL.Path)

	mediaType, params, err := mime.ParseMediaType(req.Header.Get(runtime.HeaderContentType))
	require.NoError(t, err)

	assert.Equal(t, runtime.MultipartFormMime, mediaType)
	boundary := params["boundary"]
	mr := multipart.NewReader(req.Body, boundary)
	defer req.Body.Close()
	frm, err := mr.ReadForm(1 << 20)
	require.NoError(t, err)

	assert.Equal(t, "some value", frm.Value["something"][0])
	fileverifier := func(name string, index int, filename string, content []byte) {
		mpff := frm.File[name][index]
		mpf, e := mpff.Open()
		require.NoError(t, e)
		defer mpf.Close()
		assert.Equal(t, filename, mpff.Filename)
		actual, e := io.ReadAll(mpf)
		require.NoError(t, e)
		assert.Equal(t, content, actual)
	}
	fileverifier("file", 0, "runtime.go", cont)

	fileverifier("otherfiles", 0, "runtime.go", cont)
	fileverifier("otherfiles", 1, "request.go", cont2)
	fileverifier("empty", 0, filepath.Base(emptyFile.Name()), []byte{})
}

func TestBuildRequest_BuildHTTP_Files_URLEncoded(t *testing.T) {
	cont, err := os.ReadFile("./runtime.go")
	require.NoError(t, err)
	cont2, err := os.ReadFile("./request.go")
	require.NoError(t, err)

	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetFormParam("something", "some value")
		_ = req.SetFileParam("file", mustGetFile("./runtime.go"))
		_ = req.SetFileParam("otherfiles", mustGetFile("./runtime.go"), mustGetFile("./request.go"))
		_ = req.SetQueryParam("hello", "world")
		_ = req.SetPathParam("id", "1234")
		_ = req.SetHeaderParam("X-Rate-Limit", "200")
		return nil
	})
	r := newRequest(http.MethodGet, "/flats/{id}/", reqWrtr)
	_ = r.SetHeaderParam(runtime.HeaderContentType, runtime.URLencodedFormMime)
	req, err := r.BuildHTTP(runtime.URLencodedFormMime, "", testProducers, nil)
	require.NoError(t, err)
	require.NotNil(t, req)

	assert.Equal(t, "200", req.Header.Get("x-rate-limit"))
	assert.Equal(t, "world", req.URL.Query().Get("hello"))
	assert.Equal(t, "/flats/1234/", req.URL.Path)
	mediaType, params, err := mime.ParseMediaType(req.Header.Get(runtime.HeaderContentType))
	require.NoError(t, err)

	assert.Equal(t, runtime.URLencodedFormMime, mediaType)
	boundary := params["boundary"]
	mr := multipart.NewReader(req.Body, boundary)
	defer req.Body.Close()
	frm, err := mr.ReadForm(1 << 20)
	require.NoError(t, err)

	assert.Equal(t, "some value", frm.Value["something"][0])
	fileverifier := func(name string, index int, filename string, content []byte) {
		mpff := frm.File[name][index]
		mpf, e := mpff.Open()
		require.NoError(t, e)
		defer mpf.Close()
		assert.Equal(t, filename, mpff.Filename)
		actual, e := io.ReadAll(mpf)
		require.NoError(t, e)
		assert.Equal(t, content, actual)
	}
	fileverifier("file", 0, "runtime.go", cont)

	fileverifier("otherfiles", 0, "runtime.go", cont)
	fileverifier("otherfiles", 1, "request.go", cont2)
}

type contentTypeProvider struct {
	runtime.NamedReadCloser
	contentType string
}

func (p contentTypeProvider) ContentType() string {
	return p.contentType
}

func TestBuildRequest_BuildHTTP_File_ContentType(t *testing.T) {
	cont, err := os.ReadFile("./runtime.go")
	require.NoError(t, err)
	cont2, err := os.ReadFile("./request.go")
	require.NoError(t, err)

	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetPathParam("id", "1234")
		_ = req.SetFileParam("file1", contentTypeProvider{
			NamedReadCloser: mustGetFile("./runtime.go"),
			contentType:     "application/octet-stream",
		})
		_ = req.SetFileParam("file2", mustGetFile("./request.go"))

		return nil
	})
	r := newRequest(http.MethodGet, "/flats/{id}/", reqWrtr)
	_ = r.SetHeaderParam(runtime.HeaderContentType, runtime.JSONMime)
	req, err := r.BuildHTTP(runtime.JSONMime, "", testProducers, nil)
	require.NoError(t, err)
	require.NotNil(t, req)

	assert.Equal(t, "/flats/1234/", req.URL.Path)
	mediaType, params, err := mime.ParseMediaType(req.Header.Get(runtime.HeaderContentType))
	require.NoError(t, err)
	assert.Equal(t, runtime.MultipartFormMime, mediaType)
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
		assert.Equal(t, filename, mpff.Filename)
		actual, e := io.ReadAll(mpf)
		require.NoError(t, e)
		assert.Equal(t, content, actual)
		assert.Equal(t, mpff.Header.Get("Content-Type"), contentType)
	}
	fileverifier("file1", 0, "runtime.go", cont, "application/octet-stream")
	fileverifier("file2", 0, "request.go", cont2, "text/plain; charset=utf-8")
}

func TestBuildRequest_BuildHTTP_BasePath(t *testing.T) {
	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetBodyParam(nil)
		_ = req.SetQueryParam("hello", "world")
		_ = req.SetPathParam("id", "1234")
		_ = req.SetHeaderParam("X-Rate-Limit", "200")
		return nil
	})
	r := newRequest(http.MethodPost, "/flats/{id}/", reqWrtr)

	req, err := r.BuildHTTP(runtime.JSONMime, "/basepath", testProducers, nil)
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, "200", req.Header.Get("x-rate-limit"))
	assert.Equal(t, "world", req.URL.Query().Get("hello"))
	assert.Equal(t, "/basepath/flats/1234/", req.URL.Path)
}

func TestBuildRequest_BuildHTTP_EscapedPath(t *testing.T) {
	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetBodyParam(nil)
		_ = req.SetQueryParam("hello", "world")
		_ = req.SetPathParam("id", "1234/?*&^%")
		_ = req.SetHeaderParam("X-Rate-Limit", "200")
		return nil
	})
	r := newRequest(http.MethodPost, "/flats/{id}/", reqWrtr)

	req, err := r.BuildHTTP(runtime.JSONMime, "/basepath", testProducers, nil)
	require.NoError(t, err)
	require.NotNil(t, req)

	assert.Equal(t, "200", req.Header.Get("x-rate-limit"))
	assert.Equal(t, "world", req.URL.Query().Get("hello"))
	assert.Equal(t, "/basepath/flats/1234/?*&^%/", req.URL.Path)
	assert.Equal(t, "/basepath/flats/1234%2F%3F%2A&%5E%25/", req.URL.RawPath)
	assert.Equal(t, req.URL.RawPath, req.URL.EscapedPath())
}

func TestBuildRequest_BuildHTTP_BasePathWithQueryParameters(t *testing.T) {
	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetBodyParam(nil)
		_ = req.SetQueryParam("hello", "world")
		_ = req.SetPathParam("id", "1234")
		return nil
	})
	r := newRequest(http.MethodPost, "/flats/{id}/", reqWrtr)

	req, err := r.BuildHTTP(runtime.JSONMime, "/basepath?foo=bar", testProducers, nil)
	require.NoError(t, err)
	require.NotNil(t, req)

	assert.Equal(t, "world", req.URL.Query().Get("hello"))
	assert.Equal(t, "bar", req.URL.Query().Get("foo"))
	assert.Equal(t, "/basepath/flats/1234/", req.URL.Path)
}

func TestBuildRequest_BuildHTTP_PathPatternWithQueryParameters(t *testing.T) {
	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetBodyParam(nil)
		_ = req.SetQueryParam("hello", "world")
		_ = req.SetPathParam("id", "1234")
		return nil
	})
	r := newRequest(http.MethodPost, "/flats/{id}/?foo=bar", reqWrtr)

	req, err := r.BuildHTTP(runtime.JSONMime, "/basepath", testProducers, nil)
	require.NoError(t, err)
	require.NotNil(t, req)

	assert.Equal(t, "world", req.URL.Query().Get("hello"))
	assert.Equal(t, "bar", req.URL.Query().Get("foo"))
	assert.Equal(t, "/basepath/flats/1234/", req.URL.Path)
}

func TestBuildRequest_BuildHTTP_StaticParametersPathPatternPrevails(t *testing.T) {
	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetBodyParam(nil)
		_ = req.SetPathParam("id", "1234")
		return nil
	})
	r := newRequest(http.MethodPost, "/flats/{id}/?hello=world", reqWrtr)

	req, err := r.BuildHTTP(runtime.JSONMime, "/basepath?hello=kitty", testProducers, nil)
	require.NoError(t, err)
	require.NotNil(t, req)

	assert.Equal(t, "world", req.URL.Query().Get("hello"))
	assert.Equal(t, "/basepath/flats/1234/", req.URL.Path)
}

func TestBuildRequest_BuildHTTP_StaticParametersConflictClientPrevails(t *testing.T) {
	reqWrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		_ = req.SetBodyParam(nil)
		_ = req.SetQueryParam("hello", "there")
		_ = req.SetPathParam("id", "1234")
		return nil
	})
	r := newRequest(http.MethodPost, "/flats/{id}/?hello=world", reqWrtr)

	req, err := r.BuildHTTP(runtime.JSONMime, "/basepath?hello=kitty", testProducers, nil)
	require.NoError(t, err)
	require.NotNil(t, req)

	assert.Equal(t, "there", req.URL.Query().Get("hello"))
	assert.Equal(t, "/basepath/flats/1234/", req.URL.Path)
}

type testReqFn func(*testing.T, *http.Request)

type testRoundTripper struct {
	tr          http.RoundTripper
	testFn      testReqFn
	testHarness *testing.T
}

func (t *testRoundTripper) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	t.testFn(t.testHarness, req)
	return t.tr.RoundTrip(req)
}

func TestGetBodyCallsBeforeRoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusCreated)
		_, err := rw.Write([]byte("test result"))
		require.NoError(t, err)
	}))
	defer server.Close()
	hu, err := url.Parse(server.URL)
	require.NoError(t, err)

	client := http.DefaultClient
	transport := http.DefaultTransport

	client.Transport = &testRoundTripper{
		tr:          transport,
		testHarness: t,
		testFn: func(t *testing.T, req *http.Request) {
			// Read the body once before sending the request
			body, e := req.GetBody()
			require.NoError(t, e)
			bodyContent, e := io.ReadAll(io.Reader(body))
			require.NoError(t, e)

			require.Len(t, bodyContent, int(req.ContentLength))
			require.EqualValues(t, "\"test body\"\n", string(bodyContent))

			// Read the body a second time before sending the request
			body, e = req.GetBody()
			require.NoError(t, e)
			bodyContent, e = io.ReadAll(io.Reader(body))
			require.NoError(t, e)
			require.Len(t, bodyContent, int(req.ContentLength))
			require.EqualValues(t, "\"test body\"\n", string(bodyContent))
		},
	}

	rwrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		return req.SetBodyParam("test body")
	})

	operation := &runtime.ClientOperation{
		ID:          "getSites",
		Method:      http.MethodPost,
		PathPattern: "/",
		Params:      rwrtr,
		Client:      client,
		Reader: runtime.ClientResponseReaderFunc(func(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
			if response.Code() == http.StatusCreated {
				var res string
				if e := consumer.Consume(response.Body(), &res); e != nil {
					return nil, e
				}
				return res, nil
			}
			return nil, errors.New("unexpected error code")
		}),
	}

	openAPIClient := New(hu.Host, "/", []string{schemeHTTP})
	res, err := openAPIClient.Submit(operation)
	require.NoError(t, err)

	actual := res.(string)
	require.EqualValues(t, "test result", actual)
}
