// SPDX-FileCopyrightText: Copyright 2015-2026 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"context"
	stderrors "errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"

	"github.com/go-openapi/errors"
)

// MultipartFormStreamOption configures [NewMultipartFormStream].
type MultipartFormStreamOption func(*multipartFormStreamConfig)

type multipartFormStreamConfig struct {
	maxBody        int64
	maxFiles       int
	maxFilenameLen int
}

// MultipartFormStreamMaxBody caps the total number of request-body bytes read
// by a [MultipartFormStream].
//
// A value of 0 applies [DefaultMaxUploadBodySize]. A negative value disables
// the cap when the caller has already limited the request body upstream.
func MultipartFormStreamMaxBody(n int64) MultipartFormStreamOption {
	return func(c *multipartFormStreamConfig) { c.maxBody = n }
}

// MultipartFormStreamMaxFiles rejects a multipart stream after more than n
// file parts have been encountered. A value of 0 means no file-count cap.
func MultipartFormStreamMaxFiles(n int) MultipartFormStreamOption {
	return func(c *multipartFormStreamConfig) { c.maxFiles = n }
}

// MultipartFormStreamMaxFilenameLen rejects file parts whose filename exceeds
// n bytes. A value of 0 disables the limit. When this option is not supplied,
// [DefaultMaxUploadFilenameLength] is used.
func MultipartFormStreamMaxFilenameLen(n int) MultipartFormStreamOption {
	return func(c *multipartFormStreamConfig) { c.maxFilenameLen = n }
}

// StreamedFile exposes a file part directly from the multipart request body.
//
// Reads block until bytes arrive from the client. StreamedFile is not seekable
// and is not safe for concurrent use. Its form name, filename and MIME headers
// are available before the payload is consumed.
//
// Closing a StreamedFile drains only the unread remainder of that file part.
// Close may therefore block while the client is still uploading the current
// part. The owning [MultipartFormStream] may then advance to the next part.
type StreamedFile struct {
	FieldName string
	Filename  string
	Header    textproto.MIMEHeader
	part      *multipart.Part
	closeErr  error
}

// Read reads file payload bytes directly from the request body.
func (f *StreamedFile) Read(p []byte) (int, error) {
	if f == nil || f.part == nil {
		return 0, io.ErrClosedPipe
	}

	return f.part.Read(p)
}

// Close discards the unread remainder of this file part.
//
// Close does not close the underlying HTTP request body and does not consume
// subsequent multipart parts. After Close returns successfully, the parent
// MultipartFormStream may advance to the next part.
//
// Any error encountered while discarding the unread payload is returned.
func (f *StreamedFile) Close() error {
	if f == nil {
		return nil
	}
	if f.part == nil {
		return f.closeErr
	}

	part := f.part
	f.part = nil
	// multipart.Part.Close drains with io.Copy but intentionally discards the
	// resulting error, so drain explicitly to preserve error propagation.
	_, f.closeErr = io.Copy(io.Discard, part)

	return f.closeErr
}

// MultipartFormStream reads multipart/form-data sequentially without parsing
// the complete request body before exposing file payloads.
//
// [MultipartFormStream.NextFile] consumes ordinary form fields until it reaches
// the next file part. Fields are appended to request.PostForm and request.Form
// as they are encountered. Consequently, fields after a file become visible
// only after the caller consumes or closes that file and advances the stream.
//
// A stream and its returned files are not safe for concurrent use. There is at
// most one active file part. Calling [MultipartFormStream.NextFile] closes and
// drains an unread active file before advancing, and may therefore block until
// the current part finishes arriving. No background goroutines are started.
//
// The caller owns the stream. Call [MultipartFormStream.Drain] to consume the
// remaining body, collect trailing fields and allow HTTP connection reuse when
// possible. Call [MultipartFormStream.Close] to abort immediately without
// draining.
type MultipartFormStream struct {
	request        *http.Request
	reader         *multipart.Reader
	queryValues    url.Values
	current        *StreamedFile
	maxFiles       int
	maxFilenameLen int
	files          int
	closed         bool
	done           bool
}

// NewMultipartFormStream creates a sequential multipart/form-data stream over
// r.Body.
//
// The constructor validates the media type and boundary but does not consume
// multipart parts. It initializes request.Form and request.PostForm in the same
// way as [http.Request.ParseForm], then populates multipart values incrementally
// as [MultipartFormStream.NextFile] advances.
//
// NewMultipartFormStream marks the request as handled by MultipartReader.
// Callers must not subsequently call [http.Request.ParseMultipartForm] or
// [BindForm] for the same request.
//
// File payloads are exposed directly from the request body and are not buffered
// in memory or temporary files. Ordinary form values are read into memory as
// they are encountered. Parts are processed in wire order.
//
// At most one StreamedFile may be active at a time. Calling
// [MultipartFormStream.NextFile] automatically closes and drains an unread
// current file before advancing. No background goroutines are started.
//
// MultipartFormStream is not safe for concurrent use.
func NewMultipartFormStream(r *http.Request, opts ...MultipartFormStreamOption) (*MultipartFormStream, error) {
	cfg := multipartFormStreamConfig{
		maxFilenameLen: DefaultMaxUploadFilenameLength,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	if r == nil {
		return nil, errors.NewParseError("body", "formData", "", stderrors.New("nil request"))
	}
	if r.Body == nil {
		return nil, errors.NewParseError("body", "formData", "", stderrors.New("nil request body"))
	}

	body := r.Body
	if cfg.maxBody >= 0 {
		maxBody := cfg.maxBody
		if maxBody == 0 {
			maxBody = DefaultMaxUploadBodySize
		}
		body = http.MaxBytesReader(nil, body, maxBody)
	}
	body = &contextReadCloser{ctx: r.Context(), ReadCloser: body}
	r.Body = body

	reader, err := r.MultipartReader()
	if err != nil {
		return nil, errors.NewParseError("body", "formData", "", err)
	}
	if err = r.ParseForm(); err != nil {
		return nil, errors.NewParseError("body", "formData", "", err)
	}

	return &MultipartFormStream{
		request:        r,
		reader:         reader,
		queryValues:    cloneFormValues(r.Form),
		maxFiles:       cfg.maxFiles,
		maxFilenameLen: cfg.maxFilenameLen,
	}, nil
}

// NextFile advances through the multipart body and returns the next file part.
// Ordinary form fields encountered before that file are added to request.Form
// and request.PostForm.
//
// If the previously returned file is still open, NextFile closes and drains it
// before advancing. This may block while the client is still uploading that
// part. Any drain error is returned and the stream is aborted.
//
// NextFile returns io.EOF when no file parts remain. At that point all trailing
// ordinary fields have been collected.
func (s *MultipartFormStream) NextFile() (*StreamedFile, error) {
	if s == nil {
		return nil, io.ErrClosedPipe
	}
	if s.done {
		return nil, io.EOF
	}
	if s.closed {
		return nil, io.ErrClosedPipe
	}
	if err := s.closeCurrent(); err != nil {
		return nil, s.abort(err)
	}

	for {
		part, err := s.reader.NextPart()
		if err != nil {
			if stderrors.Is(err, io.EOF) {
				s.done = true

				return nil, io.EOF
			}

			return nil, s.abort(err)
		}

		fieldName := part.FormName()
		filename := part.FileName()
		if filename == "" {
			if err := s.bindValue(part, fieldName); err != nil {
				return nil, s.abort(err)
			}

			continue
		}

		s.files++
		if s.maxFiles > 0 && s.files > s.maxFiles {
			return nil, s.abort(errors.NewParseError("body", "formData", "",
				fmt.Errorf("multipart form contains %d file parts, exceeds limit %d", s.files, s.maxFiles)))
		}
		if err := ValidateFilenameLength(fieldName, "formData", filename, s.maxFilenameLen); err != nil {
			return nil, s.abort(err)
		}

		file := &StreamedFile{
			FieldName: fieldName,
			Filename:  filename,
			Header:    part.Header,
			part:      part,
		}
		s.current = file

		return file, nil
	}
}

// Drain consumes the rest of the multipart body.
//
// Unread file payloads are discarded. Non-file form fields encountered while
// draining are collected in the request form values.
//
// Drain closes the underlying request body after reaching EOF. Subsequent calls
// to NextFile return io.EOF. Drain returns any multipart parsing, payload drain
// or request-body close error.
func (s *MultipartFormStream) Drain() error {
	if s == nil || s.closed {
		return nil
	}

	for {
		file, err := s.NextFile()
		if stderrors.Is(err, io.EOF) {
			return s.Close()
		}
		if err != nil {
			return stderrors.Join(err, s.Close())
		}
		if err := file.Close(); err != nil {
			return stderrors.Join(err, s.Close())
		}
	}
}

// Close closes the underlying HTTP request body without draining it.
//
// Close aborts further multipart processing. Call Drain instead when the
// remaining parts must be consumed, for example to collect trailing form
// fields or improve the chance of HTTP connection reuse.
func (s *MultipartFormStream) Close() error {
	if s == nil || s.closed {
		return nil
	}

	s.closed = true
	if s.current != nil {
		s.current.part = nil
		s.current = nil
	}

	return s.request.Body.Close()
}

func (s *MultipartFormStream) abort(err error) error {
	return stderrors.Join(err, s.Close())
}

func (s *MultipartFormStream) closeCurrent() error {
	if s.current == nil {
		return nil
	}

	err := s.current.Close()
	s.current = nil

	return err
}

func (s *MultipartFormStream) bindValue(part *multipart.Part, name string) error {
	defer part.Close()

	value, err := io.ReadAll(part)
	if err != nil {
		return err
	}
	if name == "" {
		return nil
	}

	s.request.PostForm.Add(name, string(value))
	postValues := s.request.PostForm[name]
	combined := make([]string, 0, len(postValues)+len(s.queryValues[name]))
	combined = append(combined, postValues...)
	combined = append(combined, s.queryValues[name]...)
	s.request.Form[name] = combined

	return nil
}

func cloneFormValues(values url.Values) url.Values {
	cloned := make(url.Values, len(values))
	for name, entries := range values {
		cloned[name] = append([]string(nil), entries...)
	}

	return cloned
}

type contextReadCloser struct {
	io.ReadCloser

	ctx context.Context //nolint:containedctx // Read has no context parameter, so the wrapper must retain it
}

func (r *contextReadCloser) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}

	return r.ReadCloser.Read(p)
}
