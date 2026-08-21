// SPDX-FileCopyrightText: Copyright 2015-2026 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"bytes"
	"context"
	stderrors "errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

const streamedFilename = "payload.bin"

func TestMultipartFormStreamExposesFileBeforeBodyCompletes(t *testing.T) {
	pipeReader, pipeWriter := io.Pipe()
	writer := multipart.NewWriter(pipeWriter)
	paused := make(chan struct{})
	resume := make(chan struct{})
	writeErr := make(chan error, 1)

	go func() {
		defer close(writeErr)
		if err := writer.WriteField("before", "value"); err != nil {
			writeErr <- err
			return
		}
		part, err := writer.CreateFormFile(testFieldFile, streamedFilename)
		if err != nil {
			writeErr <- err
			return
		}
		if _, err := io.WriteString(part, "first"); err != nil {
			writeErr <- err
			return
		}
		close(paused)
		<-resume
		if _, err := io.WriteString(part, "second"); err != nil {
			writeErr <- err
			return
		}
		if err := writer.Close(); err != nil {
			writeErr <- err
			return
		}
		if err := pipeWriter.Close(); err != nil {
			writeErr <- err
		}
	}()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testUploadPath, pipeReader)
	request.Header.Set(HeaderContentType, writer.FormDataContentType())
	stream, err := NewMultipartFormStream(request)
	require.NoError(t, err)
	defer stream.Close()

	file, err := stream.NextFile()
	require.NoError(t, err)
	assert.EqualT(t, streamedFilename, file.Filename)
	assert.EqualT(t, "value", request.Form.Get("before"))

	first := make([]byte, len("first"))
	_, err = io.ReadFull(file, first)
	require.NoError(t, err)
	assert.EqualT(t, "first", string(first))
	<-paused

	readResult := make(chan struct {
		data string
		err  error
	}, 1)
	go func() {
		data, readErr := io.ReadAll(file)
		readResult <- struct {
			data string
			err  error
		}{data: string(data), err: readErr}
	}()

	select {
	case result := <-readResult:
		t.Fatalf("file read completed while client was paused: data=%q err=%v", result.data, result.err)
	case <-time.After(100 * time.Millisecond):
	}

	close(resume)
	result := <-readResult
	require.NoError(t, result.err)
	assert.EqualT(t, "second", result.data)
	require.NoError(t, stream.Drain())
	for err := range writeErr {
		require.NoError(t, err)
	}
}

func TestMultipartFormStreamPreservesPartOrderAndFields(t *testing.T) {
	body, contentType := orderedMultipartBody(t,
		orderedField{name: "before", value: "one"},
		orderedFile{field: testFieldFile1, filename: testFileFieldA, content: "AAA"},
		orderedField{name: "between", value: "two"},
		orderedFile{field: testFieldFile2, filename: testFileFieldB, content: "BBBB"},
		orderedField{name: "after", value: "three"},
	)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testUploadPath, body)
	request.Header.Set(HeaderContentType, contentType)

	stream, err := NewMultipartFormStream(request)
	require.NoError(t, err)
	defer stream.Close()

	first, err := stream.NextFile()
	require.NoError(t, err)
	assert.EqualT(t, testFieldFile1, first.FieldName)
	assert.EqualT(t, "one", request.Form.Get("before"))
	assert.Empty(t, request.Form.Get("between"))
	assert.Equal(t, []string{"one"}, stream.Fields()["before"])
	assert.Empty(t, stream.Fields()["between"])
	firstFiles := stream.Files()
	require.Len(t, firstFiles, 1)
	assert.EqualT(t, testFieldFile1, firstFiles[0].FieldName)
	assert.EqualT(t, testFileFieldA, firstFiles[0].Filename)
	content, err := io.ReadAll(first)
	require.NoError(t, err)
	assert.EqualT(t, "AAA", string(content))

	second, err := stream.NextFile()
	require.NoError(t, err)
	assert.EqualT(t, testFieldFile2, second.FieldName)
	assert.EqualT(t, "two", request.Form.Get("between"))
	assert.Empty(t, request.Form.Get("after"))
	assert.Equal(t, []string{"one"}, stream.Fields()["before"])
	assert.Equal(t, []string{"two"}, stream.Fields()["between"])
	assert.Empty(t, stream.Fields()["after"])
	secondFiles := stream.Files()
	require.Len(t, secondFiles, 2)
	assert.EqualT(t, testFieldFile2, secondFiles[1].FieldName)
	assert.EqualT(t, testFileFieldB, secondFiles[1].Filename)
	content, err = io.ReadAll(second)
	require.NoError(t, err)
	assert.EqualT(t, "BBBB", string(content))

	_, err = stream.NextFile()
	require.ErrorIs(t, err, io.EOF)
	assert.EqualT(t, "three", request.Form.Get("after"))
	assert.Equal(t, []string{"three"}, stream.Fields()["after"])
}

func TestMultipartFormStreamDiscoverySnapshots(t *testing.T) {
	body, contentType := orderedMultipartBody(t,
		orderedField{name: "tag", value: "one"},
		orderedField{name: "tag", value: "two"},
		orderedFile{field: testFieldFile, filename: streamedFilename, content: "payload"},
	)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testUploadPath, body)
	request.Header.Set(HeaderContentType, contentType)

	stream, err := NewMultipartFormStream(request)
	require.NoError(t, err)
	defer stream.Close()

	_, err = stream.NextFile()
	require.NoError(t, err)

	fields := stream.Fields()
	assert.Equal(t, []string{"one", "two"}, fields["tag"])
	files := stream.Files()
	require.Len(t, files, 1)
	assert.EqualT(t, testFieldFile, files[0].FieldName)
	assert.EqualT(t, streamedFilename, files[0].Filename)
	assert.NotEmpty(t, files[0].Header.Get("Content-Disposition"))

	fields.Set("tag", "changed")
	files[0].Filename = "changed.bin"
	files[0].Header.Set("Content-Disposition", "changed")

	assert.Equal(t, []string{"one", "two"}, stream.Fields()["tag"])
	files = stream.Files()
	require.Len(t, files, 1)
	assert.EqualT(t, streamedFilename, files[0].Filename)
	assert.NotEqual(t, "changed", files[0].Header.Get("Content-Disposition"))
	assert.Equal(t, []string{"one", "two"}, request.PostForm["tag"])

	var nilStream *MultipartFormStream
	assert.Nil(t, nilStream.Fields())
	assert.Nil(t, nilStream.Files())
}

func TestMultipartFormStreamQueryValuesPrecedeBodyValues(t *testing.T) {
	body, contentType := orderedMultipartBody(t,
		orderedField{name: "shared", value: "body"},
		orderedFile{field: testFieldFile, filename: streamedFilename, content: "payload"},
	)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testUploadPath+"?shared=query", body)
	request.Header.Set(HeaderContentType, contentType)
	stream, err := NewMultipartFormStream(request)
	require.NoError(t, err)
	defer stream.Close()

	_, err = stream.NextFile()
	require.NoError(t, err)
	assert.Equal(t, []string{"query", "body"}, request.Form["shared"])
	assert.EqualT(t, "query", request.Form.Get("shared"))
	assert.Equal(t, []string{"body"}, stream.Fields()["shared"])
}

func TestMultipartFormStreamIgnoresPartsWithoutFormName(t *testing.T) {
	body, contentType := orderedMultipartBody(t,
		orderedRawPart{
			header: textproto.MIMEHeader{
				"Content-Disposition": {`attachment; filename="ignored.bin"`},
			},
			content: "ignored attachment",
		},
		orderedRawPart{
			header: textproto.MIMEHeader{
				"Content-Disposition": {`form-data; filename="also-ignored.bin"`},
			},
			content: "ignored unnamed form part",
		},
		orderedFile{field: testFieldFile, filename: streamedFilename, content: "payload"},
	)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testUploadPath, body)
	request.Header.Set(HeaderContentType, contentType)
	stream, err := NewMultipartFormStream(request)
	require.NoError(t, err)
	defer stream.Close()

	file, err := stream.NextFile()
	require.NoError(t, err)
	assert.EqualT(t, testFieldFile, file.FieldName)
	assert.EqualT(t, streamedFilename, file.Filename)
}

func TestMultipartFormStreamNextFileDrainsPreviousFile(t *testing.T) {
	body, contentType := orderedMultipartBody(t,
		orderedFile{field: testFieldFile1, filename: testFileFieldA, content: "AAAA"},
		orderedFile{field: testFieldFile2, filename: testFileFieldB, content: "BBBB"},
	)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testUploadPath, body)
	request.Header.Set(HeaderContentType, contentType)
	stream, err := NewMultipartFormStream(request)
	require.NoError(t, err)
	defer stream.Close()

	first, err := stream.NextFile()
	require.NoError(t, err)
	one := make([]byte, 1)
	_, err = io.ReadFull(first, one)
	require.NoError(t, err)

	second, err := stream.NextFile()
	require.NoError(t, err)
	assert.EqualT(t, testFileFieldB, second.Filename)
	_, err = first.Read(one)
	require.ErrorIs(t, err, io.ErrClosedPipe)
}

func TestStreamedFileCloseReturnsDrainError(t *testing.T) {
	pipeReader, pipeWriter := io.Pipe()
	writer := multipart.NewWriter(pipeWriter)
	wantErr := stderrors.New("injected body read failure")
	writeDone := make(chan struct{})

	go func() {
		defer close(writeDone)
		part, err := writer.CreateFormFile(testFieldFile, streamedFilename)
		if err != nil {
			_ = pipeWriter.CloseWithError(err)

			return
		}
		if _, err := io.WriteString(part, "payload"); err != nil {
			_ = pipeWriter.CloseWithError(err)

			return
		}
		_ = pipeWriter.CloseWithError(wantErr)
	}()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testUploadPath, pipeReader)
	request.Header.Set(HeaderContentType, writer.FormDataContentType())
	stream, err := NewMultipartFormStream(request)
	require.NoError(t, err)
	defer stream.Close()

	file, err := stream.NextFile()
	require.NoError(t, err)
	require.ErrorIs(t, file.Close(), wantErr)
	_, err = stream.NextFile()
	require.ErrorIs(t, err, wantErr)
	<-writeDone
}

func TestMultipartFormStreamDrainCollectsTrailingFieldsAndClosesBody(t *testing.T) {
	body, contentType := orderedMultipartBody(t,
		orderedFile{field: testFieldFile, filename: streamedFilename, content: "payload"},
		orderedField{name: "after", value: "value"},
	)
	trackedBody := &observableReadCloser{Reader: body}
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testUploadPath, nil)
	request.Body = trackedBody
	request.Header.Set(HeaderContentType, contentType)
	stream, err := NewMultipartFormStream(request)
	require.NoError(t, err)

	_, err = stream.NextFile()
	require.NoError(t, err)
	require.NoError(t, stream.Drain())
	assert.EqualT(t, "value", request.Form.Get("after"))
	assert.TrueT(t, trackedBody.Closed())
	_, err = stream.NextFile()
	require.ErrorIs(t, err, io.EOF)
}

func TestMultipartFormStreamCloseIgnoresBodyCloseEOF(t *testing.T) {
	// Since go1.27.0, a net/http server request body reports io.EOF from Close
	// when it discards the unread remainder of the request. The stream must not
	// pass that on as a failure.
	body, contentType := orderedMultipartBody(t,
		orderedFile{field: testFieldFile, filename: streamedFilename, content: "payload"},
		orderedField{name: "after", value: "value"},
	)
	trackedBody := &observableReadCloser{Reader: body, closeErr: io.EOF}
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testUploadPath, nil)
	request.Body = trackedBody
	request.Header.Set(HeaderContentType, contentType)
	stream, err := NewMultipartFormStream(request)
	require.NoError(t, err)

	_, err = stream.NextFile()
	require.NoError(t, err)
	require.NoError(t, stream.Drain())
	assert.EqualT(t, "value", request.Form.Get("after"))
	assert.TrueT(t, trackedBody.Closed())
}

func TestMultipartFormStreamCloseReportsBodyCloseError(t *testing.T) {
	body, contentType := orderedMultipartBody(t,
		orderedFile{field: testFieldFile, filename: streamedFilename, content: "payload"},
	)
	closeErr := stderrors.New("close failed")
	trackedBody := &observableReadCloser{Reader: body, closeErr: closeErr}
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testUploadPath, nil)
	request.Body = trackedBody
	request.Header.Set(HeaderContentType, contentType)
	stream, err := NewMultipartFormStream(request)
	require.NoError(t, err)

	require.ErrorIs(t, stream.Close(), closeErr)
	assert.TrueT(t, trackedBody.Closed())
}

func TestMultipartFormStreamCloseAbortsWithoutDraining(t *testing.T) {
	body, contentType := orderedMultipartBody(t,
		orderedFile{field: testFieldFile, filename: streamedFilename, content: "payload"},
		orderedField{name: "after", value: "value"},
	)
	trackedBody := &observableReadCloser{Reader: body}
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testUploadPath, nil)
	request.Body = trackedBody
	request.Header.Set(HeaderContentType, contentType)
	stream, err := NewMultipartFormStream(request)
	require.NoError(t, err)

	file, err := stream.NextFile()
	require.NoError(t, err)
	require.NoError(t, stream.Close())
	assert.TrueT(t, trackedBody.Closed())
	assert.Empty(t, request.Form.Get("after"))
	_, err = file.Read(make([]byte, 1))
	require.ErrorIs(t, err, io.ErrClosedPipe)
}

func TestMultipartFormStreamBodyLimit(t *testing.T) {
	body, contentType := orderedMultipartBody(t,
		orderedFile{field: testFieldFile, filename: streamedFilename, content: strings.Repeat("x", 1024)},
	)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testUploadPath, body)
	request.Header.Set(HeaderContentType, contentType)
	stream, err := NewMultipartFormStream(request, MultipartFormStreamMaxBody(512))
	require.NoError(t, err)
	defer stream.Close()

	file, err := stream.NextFile()
	require.NoError(t, err)
	_, err = io.ReadAll(file)
	require.Error(t, err)
	var maxBytesErr *http.MaxBytesError
	require.True(t, stderrors.As(err, &maxBytesErr), "expected *http.MaxBytesError, got %T", err)
}

func TestMultipartFormStreamPropagatesMaxBytesHandlerError(t *testing.T) {
	const limit int64 = 512

	tests := []struct {
		name    string
		consume func(*StreamedFile) error
	}{
		{
			name: "read",
			consume: func(file *StreamedFile) error {
				_, err := io.ReadAll(file)

				return err
			},
		},
		{
			name: "drain",
			consume: func(file *StreamedFile) error {
				return file.Close()
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body, contentType := orderedMultipartBody(t,
				orderedFile{
					field:    testFieldFile,
					filename: streamedFilename,
					content:  strings.Repeat("x", 1024),
				},
			)

			var consumeErr error
			handler := http.MaxBytesHandler(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
				stream, err := NewMultipartFormStream(request, MultipartFormStreamMaxBody(-1))
				require.NoError(t, err)
				defer stream.Close()

				file, err := stream.NextFile()
				require.NoError(t, err)
				consumeErr = test.consume(file)
			}), limit)

			request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testUploadPath, body)
			request.Header.Set(HeaderContentType, contentType)
			handler.ServeHTTP(httptest.NewRecorder(), request)

			var maxBytesErr *http.MaxBytesError
			require.True(t, stderrors.As(consumeErr, &maxBytesErr),
				"expected *http.MaxBytesError, got %T", consumeErr)
			assert.EqualT(t, limit, maxBytesErr.Limit)
		})
	}
}

func TestMultipartFormStreamMalformedBody(t *testing.T) {
	body, contentType := orderedMultipartBody(t,
		orderedFile{field: testFieldFile, filename: streamedFilename, content: "payload"},
	)
	truncated := body.Bytes()[:body.Len()-5]
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testUploadPath, bytes.NewReader(truncated))
	request.Header.Set(HeaderContentType, contentType)
	stream, err := NewMultipartFormStream(request)
	require.NoError(t, err)
	defer stream.Close()

	file, err := stream.NextFile()
	require.NoError(t, err)
	_, err = io.ReadAll(file)
	require.Error(t, err)
}

func TestMultipartFormStreamContextCancellation(t *testing.T) {
	body, contentType := orderedMultipartBody(t,
		orderedFile{field: testFieldFile, filename: streamedFilename, content: "payload"},
	)
	ctx, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequestWithContext(ctx, http.MethodPost, testUploadPath, body)
	request.Header.Set(HeaderContentType, contentType)
	stream, err := NewMultipartFormStream(request)
	require.NoError(t, err)
	defer stream.Close()

	cancel()
	_, err = stream.NextFile()
	require.ErrorIs(t, err, context.Canceled)
}

func TestMultipartFormStreamLimits(t *testing.T) {
	t.Run("filename", func(t *testing.T) {
		body, contentType := orderedMultipartBody(t,
			orderedFile{field: testFieldFile, filename: "too-long.txt", content: "payload"},
		)
		request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testUploadPath, body)
		request.Header.Set(HeaderContentType, contentType)
		stream, err := NewMultipartFormStream(request, MultipartFormStreamMaxFilenameLen(4))
		require.NoError(t, err)
		defer stream.Close()

		_, err = stream.NextFile()
		var parseErr *errors.ParseError
		require.True(t, stderrors.As(err, &parseErr), "expected *errors.ParseError, got %T", err)
		assert.EqualT(t, testFieldFile, parseErr.Name)
	})

	t.Run("parts", func(t *testing.T) {
		body, contentType := orderedMultipartBody(t,
			orderedField{name: "first", value: "one"},
			orderedField{name: "second", value: "two"},
			orderedFile{field: testFieldFile, filename: streamedFilename, content: "payload"},
		)
		request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testUploadPath, body)
		request.Header.Set(HeaderContentType, contentType)
		stream, err := NewMultipartFormStream(request, MultipartFormStreamMaxParts(1))
		require.NoError(t, err)
		defer stream.Close()

		_, err = stream.NextFile()
		var parseErr *errors.ParseError
		require.True(t, stderrors.As(err, &parseErr), "expected *errors.ParseError, got %T", err)
	})

	t.Run("files", func(t *testing.T) {
		body, contentType := orderedMultipartBody(t,
			orderedFile{field: testFieldFile1, filename: testFileFieldA, content: "A"},
			orderedFile{field: testFieldFile2, filename: testFileFieldB, content: "B"},
		)
		request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testUploadPath, body)
		request.Header.Set(HeaderContentType, contentType)
		stream, err := NewMultipartFormStream(request, MultipartFormStreamMaxFiles(1))
		require.NoError(t, err)
		defer stream.Close()

		first, err := stream.NextFile()
		require.NoError(t, err)
		_, err = io.ReadAll(first)
		require.NoError(t, err)
		_, err = stream.NextFile()
		var parseErr *errors.ParseError
		require.True(t, stderrors.As(err, &parseErr), "expected *errors.ParseError, got %T", err)
	})
}

func TestNewMultipartFormStreamReturnsEmptyForUnsupportedMethods(t *testing.T) {
	methods := []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodDelete,
		http.MethodOptions,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			request := httptest.NewRequestWithContext(
				t.Context(),
				method,
				testUploadPath+"?query=value",
				strings.NewReader("this body must not be parsed as multipart"),
			)
			request.Header.Set(HeaderContentType, URLencodedFormMime)

			stream, err := NewMultipartFormStream(request)
			require.NoError(t, err)
			defer stream.Close()

			_, err = stream.NextFile()
			require.ErrorIs(t, err, io.EOF)
			assert.EqualT(t, "value", request.Form.Get("query"))
			assert.Empty(t, request.PostForm)
		})
	}
}

func TestNewMultipartFormStreamRejectsNonMultipartRequest(t *testing.T) {
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testUploadPath, strings.NewReader("value=x"))
	request.Header.Set(HeaderContentType, URLencodedFormMime)

	_, err := NewMultipartFormStream(request)
	var parseErr *errors.ParseError
	require.True(t, stderrors.As(err, &parseErr), "expected *errors.ParseError, got %T", err)
}

func TestNewMultipartFormStreamRejectsMultipartMixed(t *testing.T) {
	body, contentType := orderedMultipartBody(t,
		orderedFile{field: testFieldFile, filename: streamedFilename, content: "payload"},
	)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testUploadPath, body)
	request.Header.Set(HeaderContentType, strings.Replace(contentType, MultipartFormMime, "multipart/mixed", 1))

	_, err := NewMultipartFormStream(request)
	var parseErr *errors.ParseError
	require.True(t, stderrors.As(err, &parseErr), "expected *errors.ParseError, got %T", err)
	require.ErrorIs(t, parseErr.Reason, http.ErrNotMultipart)
}

func TestNewMultipartFormStreamRejectsEmptyBoundary(t *testing.T) {
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testUploadPath, strings.NewReader("body"))
	request.Header.Set(HeaderContentType, `multipart/form-data; boundary=""`)

	_, err := NewMultipartFormStream(request)
	var parseErr *errors.ParseError
	require.True(t, stderrors.As(err, &parseErr), "expected *errors.ParseError, got %T", err)
	require.ErrorIs(t, parseErr.Reason, http.ErrMissingBoundary)
}

type orderedMultipartPart interface {
	writeTo(*testing.T, *multipart.Writer)
}

type orderedRawPart struct {
	header  textproto.MIMEHeader
	content string
}

func (p orderedRawPart) writeTo(t *testing.T, writer *multipart.Writer) {
	t.Helper()
	part, err := writer.CreatePart(p.header)
	require.NoError(t, err)
	_, err = io.WriteString(part, p.content)
	require.NoError(t, err)
}

type orderedField struct {
	name  string
	value string
}

func (f orderedField) writeTo(t *testing.T, writer *multipart.Writer) {
	t.Helper()
	require.NoError(t, writer.WriteField(f.name, f.value))
}

type orderedFile struct {
	field    string
	filename string
	content  string
}

func (f orderedFile) writeTo(t *testing.T, writer *multipart.Writer) {
	t.Helper()
	part, err := writer.CreateFormFile(f.field, f.filename)
	require.NoError(t, err)
	_, err = io.WriteString(part, f.content)
	require.NoError(t, err)
}

func orderedMultipartBody(t *testing.T, parts ...orderedMultipartPart) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for _, part := range parts {
		part.writeTo(t, writer)
	}
	require.NoError(t, writer.Close())

	return body, writer.FormDataContentType()
}

type observableReadCloser struct {
	io.Reader

	closeErr error

	mu     sync.Mutex
	closed bool
}

func (r *observableReadCloser) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true

	return r.closeErr
}

func (r *observableReadCloser) Closed() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.closed
}
