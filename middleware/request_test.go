// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

const (
	csvFormat = "csv"
	testURL   = "http://localhost:8002/hello"
)

type stubConsumer struct {
}

func (s *stubConsumer) Consume(_ io.Reader, _ any) error {
	return nil
}

type friend struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type jsonRequestParams struct {
	ID        int64    // path
	Name      string   // query
	Friend    friend   // body
	RequestID int64    // header
	Tags      []string // csv
}

type jsonRequestPtr struct {
	ID        int64    // path
	Name      string   // query
	RequestID int64    // header
	Tags      []string // csv
	Friend    *friend
}

type jsonRequestSlice struct {
	ID        int64    // path
	Name      string   // query
	RequestID int64    // header
	Tags      []string // csv
	Friend    []friend
}

func parametersForAllTypes(fmt string) map[string]spec.Parameter {
	if fmt == "" {
		fmt = csvFormat
	}
	nameParam := spec.QueryParam(paramKeyName).Typed(typeString, "")
	idParam := spec.PathParam(paramKeyID).Typed("integer", "int64")
	ageParam := spec.QueryParam(paramKeyAge).Typed("integer", "int32")
	scoreParam := spec.QueryParam("score").Typed("number", "float")
	factorParam := spec.QueryParam("factor").Typed("number", "double")

	friendSchema := new(spec.Schema).Typed("object", "")
	friendParam := spec.BodyParam("friend", friendSchema)

	requestIDParam := spec.HeaderParam("X-Request-Id").Typed("integer", "int64")
	requestIDParam.Extensions = spec.Extensions(map[string]any{})
	requestIDParam.Extensions.Add("go-name", keyRequestID)

	items := new(spec.Items)
	items.Type = typeString
	tagsParam := spec.QueryParam("tags").CollectionOf(items, fmt)

	confirmedParam := spec.QueryParam("confirmed").Typed("boolean", "")
	plannedParam := spec.QueryParam("planned").Typed(typeString, "date")
	deliveredParam := spec.QueryParam("delivered").Typed(typeString, "date-time")
	pictureParam := spec.QueryParam("picture").Typed(typeString, "byte") // base64 encoded during transport

	return map[string]spec.Parameter{
		keyID:        *idParam,
		keyName:      *nameParam,
		keyRequestID: *requestIDParam,
		keyFriend:    *friendParam,
		keyTags:      *tagsParam,
		"Age":        *ageParam,
		"Score":      *scoreParam,
		"Factor":     *factorParam,
		"Confirmed":  *confirmedParam,
		"Planned":    *plannedParam,
		"Delivered":  *deliveredParam,
		"Picture":    *pictureParam,
	}
}

func parametersForJSONRequestParams(fmt string) map[string]spec.Parameter {
	if fmt == "" {
		fmt = csvFormat
	}
	nameParam := spec.QueryParam(paramKeyName).Typed(typeString, "")
	idParam := spec.PathParam(paramKeyID).Typed("integer", "int64")

	friendSchema := new(spec.Schema).Typed("object", "")
	friendParam := spec.BodyParam("friend", friendSchema)

	requestIDParam := spec.HeaderParam("X-Request-Id").Typed("integer", "int64")
	requestIDParam.Extensions = spec.Extensions(map[string]any{})
	requestIDParam.Extensions.Add("go-name", keyRequestID)

	items := new(spec.Items)
	items.Type = typeString
	tagsParam := spec.QueryParam("tags").CollectionOf(items, fmt)

	return map[string]spec.Parameter{
		keyID:        *idParam,
		keyName:      *nameParam,
		keyRequestID: *requestIDParam,
		keyFriend:    *friendParam,
		keyTags:      *tagsParam,
	}
}
func parametersForJSONRequestSliceParams(fmt string) map[string]spec.Parameter {
	if fmt == "" {
		fmt = csvFormat
	}
	nameParam := spec.QueryParam(paramKeyName).Typed(typeString, "")
	idParam := spec.PathParam(paramKeyID).Typed("integer", "int64")

	friendSchema := new(spec.Schema).Typed("object", "")
	friendParam := spec.BodyParam("friend", spec.ArrayProperty(friendSchema))

	requestIDParam := spec.HeaderParam("X-Request-Id").Typed("integer", "int64")
	requestIDParam.Extensions = spec.Extensions(map[string]any{})
	requestIDParam.Extensions.Add("go-name", keyRequestID)

	items := new(spec.Items)
	items.Type = typeString
	tagsParam := spec.QueryParam("tags").CollectionOf(items, fmt)

	return map[string]spec.Parameter{
		keyID:        *idParam,
		keyName:      *nameParam,
		keyRequestID: *requestIDParam,
		keyFriend:    *friendParam,
		keyTags:      *tagsParam,
	}
}

func TestRequestBindingDefaultValue(t *testing.T) {
	confirmed := true
	name := "thomas"
	friend := map[string]any{paramKeyName: valToby, paramKeyAge: float64(32)}
	id, age, score, factor := int64(7575), int32(348), float32(5.309), float64(37.403)
	requestID := 19394858
	tags := []string{tagOne, tagTwo, tagThree}
	dt1 := time.Date(2014, 8, 9, 0, 0, 0, 0, time.UTC)
	planned := strfmt.Date(dt1)
	dt2 := time.Date(2014, 10, 12, 8, 5, 5, 0, time.UTC)
	delivered := strfmt.DateTime(dt2)
	uri, err := url.Parse(testURL)
	require.NoError(t, err)
	defaults := map[string]any{
		paramKeyID:     id,
		paramKeyAge:    age,
		"score":        score,
		"factor":       factor,
		paramKeyName:   name,
		"friend":       friend,
		"X-Request-Id": requestID,
		"tags":         tags,
		"confirmed":    confirmed,
		"planned":      planned,
		"delivered":    delivered,
		"picture":      []byte("hello"),
	}
	op2 := parametersForAllTypes("")
	op3 := make(map[string]spec.Parameter)
	for k, p := range op2 {
		p.Default = defaults[p.Name]
		op3[k] = p
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, uri.String(), bytes.NewBuffer(nil))
	require.NoError(t, err)
	req.Header.Set("Content-Type", jsonMime)
	binder := NewUntypedRequestBinder(op3, new(spec.Swagger), strfmt.Default)

	data := make(map[string]any)
	err = binder.Bind(req, RouteParams(nil), runtime.JSONConsumer(), &data)
	require.NoError(t, err)
	assert.Equal(t, defaults[paramKeyID], data[paramKeyID])
	assert.Equal(t, name, data[paramKeyName])
	assert.Equal(t, friend, data["friend"])
	assert.EqualValues(t, requestID, data["X-Request-Id"])
	assert.Equal(t, tags, data["tags"])
	assert.Equal(t, planned, data["planned"])
	assert.Equal(t, delivered, data["delivered"])
	assert.Equal(t, confirmed, data["confirmed"])
	assert.Equal(t, age, data[paramKeyAge])
	assert.InDelta(t, factor, data["factor"], 1e-6)
	assert.InDelta(t, score, data["score"], 1e-6)
	formatted, ok := data["picture"].(strfmt.Base64)
	require.TrueT(t, ok)
	assert.EqualT(t, "hello", string(formatted))
}

func TestRequestBindingForInvalid(t *testing.T) {
	invalidParam := spec.QueryParam("some")

	op1 := map[string]spec.Parameter{"Some": *invalidParam}

	binder := NewUntypedRequestBinder(op1, new(spec.Swagger), strfmt.Default)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://localhost:8002/hello?name=the-name", nil)
	require.NoError(t, err)

	err = binder.Bind(req, nil, new(stubConsumer), new(jsonRequestParams))
	require.Error(t, err)

	op2 := parametersForJSONRequestParams("")
	binder = NewUntypedRequestBinder(op2, new(spec.Swagger), strfmt.Default)

	req, err = http.NewRequestWithContext(context.Background(), http.MethodPost, "http://localhost:8002/hello/1?name=the-name", bytes.NewBufferString(`{"name":"toby","age":32}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application(")
	data := jsonRequestParams{}
	err = binder.Bind(req, RouteParams([]RouteParam{{paramKeyID, "1"}}), runtime.JSONConsumer(), &data)
	require.Error(t, err)

	req, err = http.NewRequestWithContext(context.Background(), http.MethodPost, "http://localhost:8002/hello/1?name=the-name", bytes.NewBufferString(`{]`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", jsonMime)
	data = jsonRequestParams{}
	err = binder.Bind(req, RouteParams([]RouteParam{{paramKeyID, "1"}}), runtime.JSONConsumer(), &data)
	require.Error(t, err)

	invalidMultiParam := spec.HeaderParam("tags").CollectionOf(new(spec.Items), multiFmt)
	op3 := map[string]spec.Parameter{keyTags: *invalidMultiParam}
	binder = NewUntypedRequestBinder(op3, new(spec.Swagger), strfmt.Default)

	req, err = http.NewRequestWithContext(context.Background(), http.MethodPost, "http://localhost:8002/hello/1?name=the-name", bytes.NewBufferString(`{}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", jsonMime)
	data = jsonRequestParams{}
	err = binder.Bind(req, RouteParams([]RouteParam{{paramKeyID, "1"}}), runtime.JSONConsumer(), &data)
	require.Error(t, err)

	invalidMultiParam = spec.PathParam("").CollectionOf(new(spec.Items), multiFmt)

	op4 := map[string]spec.Parameter{keyTags: *invalidMultiParam}
	binder = NewUntypedRequestBinder(op4, new(spec.Swagger), strfmt.Default)

	req, err = http.NewRequestWithContext(context.Background(), http.MethodPost, "http://localhost:8002/hello/1?name=the-name", bytes.NewBufferString(`{}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", jsonMime)
	data = jsonRequestParams{}
	err = binder.Bind(req, RouteParams([]RouteParam{{paramKeyID, "1"}}), runtime.JSONConsumer(), &data)
	require.Error(t, err)

	invalidInParam := spec.HeaderParam("tags").Typed(typeString, "")
	invalidInParam.In = "invalid"
	op5 := map[string]spec.Parameter{keyTags: *invalidInParam}
	binder = NewUntypedRequestBinder(op5, new(spec.Swagger), strfmt.Default)

	req, err = http.NewRequestWithContext(context.Background(), http.MethodPost, "http://localhost:8002/hello/1?name=the-name", bytes.NewBufferString(`{}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", jsonMime)
	data = jsonRequestParams{}
	err = binder.Bind(req, RouteParams([]RouteParam{{paramKeyID, "1"}}), runtime.JSONConsumer(), &data)
	require.Error(t, err)
}

func TestRequestBindingForValid(t *testing.T) {
	for _, fmt := range []string{csvFormat, pipesFmt, tsvFmt, ssvFmt, multiFmt} {
		op1 := parametersForJSONRequestParams(fmt)

		binder := NewUntypedRequestBinder(op1, new(spec.Swagger), strfmt.Default)

		lval := []string{tagOne, tagTwo, tagThree}
		var queryString string
		var skipEscape bool
		switch fmt {
		case multiFmt:
			skipEscape = true
			queryString = strings.Join(lval, "&tags=")
		case ssvFmt:
			queryString = strings.Join(lval, " ")
		case pipesFmt:
			queryString = strings.Join(lval, "|")
		case tsvFmt:
			queryString = strings.Join(lval, "\t")
		default:
			queryString = strings.Join(lval, ",")
		}
		if !skipEscape {
			queryString = url.QueryEscape(queryString)
		}

		urlStr := "http://localhost:8002/hello/1?name=the-name&tags=" + queryString

		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, urlStr, bytes.NewBufferString(`{"name":"toby","age":32}`))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json;charset=utf-8")
		req.Header.Set("X-Request-Id", "1325959595")

		data := jsonRequestParams{}
		err = binder.Bind(req, RouteParams([]RouteParam{{paramKeyID, "1"}}), runtime.JSONConsumer(), &data)

		expected := jsonRequestParams{
			ID:        1,
			Name:      "the-name",
			Friend:    friend{valToby, 32},
			RequestID: 1325959595,
			Tags:      []string{tagOne, tagTwo, tagThree},
		}
		require.NoError(t, err)
		assert.Equal(t, expected, data)
	}

	op1 := parametersForJSONRequestParams("")

	binder := NewUntypedRequestBinder(op1, new(spec.Swagger), strfmt.Default)
	urlStr := "http://localhost:8002/hello/1?name=the-name&tags=one,two,three"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, urlStr, bytes.NewBufferString(`{"name":"toby","age":32}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	req.Header.Set("X-Request-Id", "1325959595")

	data2 := jsonRequestPtr{}
	err = binder.Bind(req, []RouteParam{{paramKeyID, "1"}}, runtime.JSONConsumer(), &data2)

	expected2 := jsonRequestPtr{
		Friend: &friend{valToby, 32},
		Tags:   []string{tagOne, tagTwo, tagThree},
	}
	require.NoError(t, err)
	if data2.Friend == nil {
		t.Fatal("friend is nil")
	}
	assert.EqualT(t, *expected2.Friend, *data2.Friend)
	assert.Equal(t, expected2.Tags, data2.Tags)

	req, err = http.NewRequestWithContext(context.Background(), http.MethodPost, urlStr, bytes.NewBufferString(`[{"name":"toby","age":32}]`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	req.Header.Set("X-Request-Id", "1325959595")
	op2 := parametersForJSONRequestSliceParams("")
	binder = NewUntypedRequestBinder(op2, new(spec.Swagger), strfmt.Default)
	data3 := jsonRequestSlice{}
	err = binder.Bind(req, []RouteParam{{paramKeyID, "1"}}, runtime.JSONConsumer(), &data3)

	expected3 := jsonRequestSlice{
		Friend: []friend{{valToby, 32}},
		Tags:   []string{tagOne, tagTwo, tagThree},
	}
	require.NoError(t, err)
	assert.Equal(t, expected3.Friend, data3.Friend)
	assert.Equal(t, expected3.Tags, data3.Tags)
}

type formRequest struct {
	Name string
	Age  int
}

func parametersForFormUpload() map[string]spec.Parameter {
	nameParam := spec.FormDataParam(paramKeyName).Typed(typeString, "")

	ageParam := spec.FormDataParam(paramKeyAge).Typed("integer", "int32")

	return map[string]spec.Parameter{keyName: *nameParam, "Age": *ageParam}
}

func TestFormUpload(t *testing.T) {
	params := parametersForFormUpload()
	binder := NewUntypedRequestBinder(params, new(spec.Swagger), strfmt.Default)

	urlStr := testURL
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, urlStr, bytes.NewBufferString(`name=the-name&age=32`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	data := formRequest{}
	res := binder.Bind(req, nil, runtime.JSONConsumer(), &data)
	require.NoError(t, res)
	assert.EqualT(t, "the-name", data.Name)
	assert.EqualT(t, 32, data.Age)

	req, err = http.NewRequestWithContext(context.Background(), http.MethodPost, urlStr, bytes.NewBufferString(`name=%3&age=32`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	data = formRequest{}
	require.Error(t, binder.Bind(req, nil, runtime.JSONConsumer(), &data))
}

type fileRequest struct {
	Name string       // body
	File runtime.File // upload
}

func paramsForFileUpload() *UntypedRequestBinder {
	nameParam := spec.FormDataParam(paramKeyName).Typed(typeString, "")

	fileParam := spec.FileParam("file").AsRequired()

	params := map[string]spec.Parameter{keyName: *nameParam, "File": *fileParam}
	return NewUntypedRequestBinder(params, new(spec.Swagger), strfmt.Default)
}

func TestBindingFileUpload(t *testing.T) {
	binder := paramsForFileUpload()

	body := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "plain-jane.txt")
	require.NoError(t, err)

	_, err = part.Write([]byte("the file contents"))
	require.NoError(t, err)
	require.NoError(t, writer.WriteField(paramKeyName, "the-name"))
	require.NoError(t, writer.Close())

	urlStr := testURL
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, urlStr, body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	data := fileRequest{}
	require.NoError(t, binder.Bind(req, nil, runtime.JSONConsumer(), &data))
	assert.EqualT(t, "the-name", data.Name)
	assert.NotNil(t, data.File)
	assert.NotNil(t, data.File.Header)
	assert.EqualT(t, "plain-jane.txt", data.File.Header.Filename)

	bb, err := io.ReadAll(data.File.Data)
	require.NoError(t, err)
	assert.Equal(t, []byte("the file contents"), bb)

	req, err = http.NewRequestWithContext(context.Background(), http.MethodPost, urlStr, body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", jsonMime)
	data = fileRequest{}
	require.Error(t, binder.Bind(req, nil, runtime.JSONConsumer(), &data))

	req, err = http.NewRequestWithContext(context.Background(), http.MethodPost, urlStr, body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application(")
	data = fileRequest{}
	require.Error(t, binder.Bind(req, nil, runtime.JSONConsumer(), &data))

	body = bytes.NewBuffer(nil)
	writer = multipart.NewWriter(body)
	part, err = writer.CreateFormFile("bad-name", "plain-jane.txt")
	require.NoError(t, err)

	_, err = part.Write([]byte("the file contents"))
	require.NoError(t, err)
	require.NoError(t, writer.WriteField(paramKeyName, "the-name"))
	require.NoError(t, writer.Close())
	req, err = http.NewRequestWithContext(context.Background(), http.MethodPost, urlStr, body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	data = fileRequest{}
	require.Error(t, binder.Bind(req, nil, runtime.JSONConsumer(), &data))

	req, err = http.NewRequestWithContext(context.Background(), http.MethodPost, urlStr, body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	_, err = req.MultipartReader()
	require.NoError(t, err)

	data = fileRequest{}
	require.Error(t, binder.Bind(req, nil, runtime.JSONConsumer(), &data))

	writer = multipart.NewWriter(body)
	require.NoError(t, writer.WriteField(paramKeyName, "the-name"))
	require.NoError(t, writer.Close())

	req, err = http.NewRequestWithContext(context.Background(), http.MethodPost, urlStr, body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	data = fileRequest{}
	require.Error(t, binder.Bind(req, nil, runtime.JSONConsumer(), &data))
}

func paramsForOptionalFileUpload() *UntypedRequestBinder {
	nameParam := spec.FormDataParam(paramKeyName).Typed(typeString, "")
	fileParam := spec.FileParam("file").AsOptional()

	params := map[string]spec.Parameter{keyName: *nameParam, "File": *fileParam}
	return NewUntypedRequestBinder(params, new(spec.Swagger), strfmt.Default)
}

func TestBindingOptionalFileUpload(t *testing.T) {
	binder := paramsForOptionalFileUpload()

	body := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(body)
	require.NoError(t, writer.WriteField(paramKeyName, "the-name"))
	require.NoError(t, writer.Close())

	urlStr := testURL
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, urlStr, body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	data := fileRequest{}
	require.NoError(t, binder.Bind(req, nil, runtime.JSONConsumer(), &data))
	assert.EqualT(t, "the-name", data.Name)
	assert.Nil(t, data.File.Data)
	assert.Nil(t, data.File.Header)

	writer = multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "plain-jane.txt")
	require.NoError(t, err)
	_, err = part.Write([]byte("the file contents"))
	require.NoError(t, err)
	require.NoError(t, writer.WriteField(paramKeyName, "the-name"))
	require.NoError(t, writer.Close())

	req, err = http.NewRequestWithContext(context.Background(), http.MethodPost, urlStr, body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	require.NoError(t, writer.Close())

	data = fileRequest{}
	require.NoError(t, binder.Bind(req, nil, runtime.JSONConsumer(), &data))
	assert.EqualT(t, "the-name", data.Name)
	assert.NotNil(t, data.File)
	assert.NotNil(t, data.File.Header)
	assert.EqualT(t, "plain-jane.txt", data.File.Header.Filename)

	bb, err := io.ReadAll(data.File.Data)
	require.NoError(t, err)
	assert.Equal(t, []byte("the file contents"), bb)
}

// TestBindingOptionalFileUpload_URLEncoded is the untyped-path
// regression for go-swagger/go-swagger#3113: a spec accepting both
// multipart/form-data and application/x-www-form-urlencoded with an
// optional file field must accept a urlencoded request body that omits
// the file value.
func TestBindingOptionalFileUpload_URLEncoded(t *testing.T) {
	binder := paramsForOptionalFileUpload()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, testURL, bytes.NewBufferString(`name=the-name`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	data := fileRequest{}
	require.NoError(t, binder.Bind(req, nil, runtime.JSONConsumer(), &data))
	assert.EqualT(t, "the-name", data.Name)
	assert.Nil(t, data.File.Data)
	assert.Nil(t, data.File.Header)
}

// TestBindingFileUpload_URLEncoded exercises the OpenAPI 2.0 allowance
// that file params can be consumed as application/x-www-form-urlencoded.
// The file bytes ride as a regular form value, surfaced through the
// runtime.File target with Header.Filename set to the field name.
func TestBindingFileUpload_URLEncoded(t *testing.T) {
	binder := paramsForFileUpload()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, testURL,
		bytes.NewBufferString(`name=the-name&file=the+file+contents`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	data := fileRequest{}
	require.NoError(t, binder.Bind(req, nil, runtime.JSONConsumer(), &data))
	assert.EqualT(t, "the-name", data.Name)
	require.NotNil(t, data.File.Header)
	assert.EqualT(t, "file", data.File.Header.Filename)
	assert.EqualT(t, int64(len("the file contents")), data.File.Header.Size)
	bb, err := io.ReadAll(data.File.Data)
	require.NoError(t, err)
	assert.EqualT(t, "the file contents", string(bb))
}

// TestBindingFileUpload_URLEncoded_RequiredMissing verifies that the
// untyped path produces a parse error (rather than the misleading
// multipart-parse error) when a required file field is absent from
// a urlencoded body.
func TestBindingFileUpload_URLEncoded_RequiredMissing(t *testing.T) {
	binder := paramsForFileUpload()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, testURL,
		bytes.NewBufferString(`name=the-name`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	data := fileRequest{}
	bindErr := binder.Bind(req, nil, runtime.JSONConsumer(), &data)
	require.Error(t, bindErr)
	assert.Contains(t, bindErr.Error(), http.ErrMissingFile.Error())
}

// TestBindingFileUpload_RejectsOversizedFilename exercises the
// filename-length cap on the untyped formData path: a multipart
// body with a multi-MB filename must be rejected with a ParseError
// before the file is bound.
//
// Mirrors the BindFormFile-path coverage in
// runtime.TestBindForm_maxFilenameLen_exceeded. Security scrub
// Lens 3 / L3.1.
func TestBindingFileUpload_RejectsOversizedFilename(t *testing.T) {
	binder := paramsForFileUpload()

	body := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(body)
	longName := strings.Repeat("x", runtime.DefaultMaxUploadFilenameLength+1) + ".txt"
	part, err := writer.CreateFormFile("file", longName)
	require.NoError(t, err)
	_, err = part.Write([]byte("contents"))
	require.NoError(t, err)
	require.NoError(t, writer.WriteField(paramKeyName, "the-name"))
	require.NoError(t, writer.Close())

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, testURL, body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	data := fileRequest{}
	bindErr := binder.Bind(req, nil, runtime.JSONConsumer(), &data)
	require.Error(t, bindErr)
	assert.Contains(t, bindErr.Error(), "exceeds limit")
	// File must NOT have been bound past the cap.
	assert.Nil(t, data.File.Data)
	assert.Nil(t, data.File.Header)
}
