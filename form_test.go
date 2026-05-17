// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"bytes"
	stderrors "errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

const (
	testUploadPath  = "/upload"
	testFileFieldB  = "b.txt"
	testFileFieldOK = "ok.txt"
	testFileFieldA  = "a.txt"
	testFileData    = "data"
	testFieldDesc   = "desc"
	testFieldFile   = "file"
	testFieldFile1  = "file1"
	testFieldFile2  = "file2"
	testValueHello  = "hello"
)

// multipartBody builds a multipart/form-data body with the given file
// parts ({name, filename, content}) and form values. Returns the body
// bytes and the Content-Type header to set on the request.
type multipartFile struct {
	field    string
	filename string
	content  string
}

func multipartBody(t *testing.T, files []multipartFile, values map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)
	for _, f := range files {
		fw, err := w.CreateFormFile(f.field, f.filename)
		require.NoError(t, err)
		_, err = io.WriteString(fw, f.content)
		require.NoError(t, err)
	}
	for k, v := range values {
		require.NoError(t, w.WriteField(k, v))
	}
	require.NoError(t, w.Close())
	return buf, w.FormDataContentType()
}

func newMultipartRequest(t *testing.T, files []multipartFile, values map[string]string) *http.Request {
	t.Helper()
	body, ct := multipartBody(t, files, values)
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testUploadPath, body)
	r.Header.Set("Content-Type", ct)
	return r
}

func newURLEncodedRequest(t *testing.T, values map[string]string) *http.Request {
	t.Helper()
	form := make([]string, 0, len(values))
	for k, v := range values {
		form = append(form, k+"="+v)
	}
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testUploadPath, strings.NewReader(strings.Join(form, "&")))
	r.Header.Set("Content-Type", URLencodedFormMime)
	return r
}

// assertParseError extracts a *errors.ParseError from err and checks
// its name/in fields plus an optional reason predicate.
func assertParseError(t *testing.T, err error, wantName string, reasonCheck func(error) bool) {
	t.Helper()
	require.Error(t, err)
	var pe *errors.ParseError
	require.True(t, stderrors.As(err, &pe), "expected *errors.ParseError, got %T", err)
	assert.EqualT(t, wantName, pe.Name)
	assert.EqualT(t, "formData", pe.In)
	if reasonCheck != nil {
		require.True(t, reasonCheck(pe.Reason), "reason check failed for %v", pe.Reason)
	}
}

// assertCompositeContains extracts a *errors.CompositeError from err
// and asserts that at least n inner errors satisfy match.
//
//nolint:unparam // left variable n for future assertions
func assertCompositeContains(t *testing.T, err error, n int, match func(error) bool) {
	t.Helper()
	require.Error(t, err)
	var ce *errors.CompositeError
	require.True(t, stderrors.As(err, &ce), "expected *errors.CompositeError, got %T", err)
	var got int
	for _, e := range ce.Errors {
		if match(e) {
			got++
		}
	}
	assert.EqualT(t, n, got, "matched %d inner errors, want %d", got, n)
}

func TestBindForm_parseOnly_multipart(t *testing.T) {
	r := newMultipartRequest(t, nil, map[string]string{testFieldDesc: testValueHello})

	fatal, err := BindForm(r)

	assert.FalseT(t, fatal)
	require.NoError(t, err)
	require.NotNil(t, r.MultipartForm)
	assert.EqualT(t, testValueHello, r.Form.Get(testFieldDesc))
}

func TestBindForm_parseOnly_urlencoded(t *testing.T) {
	r := newURLEncodedRequest(t, map[string]string{testFieldDesc: testValueHello, "count": "42"})

	fatal, err := BindForm(r)

	assert.FalseT(t, fatal)
	require.NoError(t, err)
	assert.EqualT(t, testValueHello, r.PostForm.Get(testFieldDesc))
	assert.EqualT(t, "42", r.PostForm.Get("count"))
}

func TestBindForm_parseFailure_urlencoded(t *testing.T) {
	// Malformed URL escape in an urlencoded body. An earlier draft routed
	// through ParseMultipartForm which silently swallowed this class of
	// errors; this test guards against the regression.
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testUploadPath, strings.NewReader("name=%3&age=32"))
	r.Header.Set("Content-Type", URLencodedFormMime)

	fatal, err := BindForm(r)

	assert.TrueT(t, fatal)
	assertParseError(t, err, "body", nil)
}

func TestBindForm_parseFailure(t *testing.T) {
	// Multipart Content-Type but truncated body — ParseMultipartForm fails.
	body, ct := multipartBody(t, []multipartFile{{testFieldFile, "f.txt", testValueHello}}, nil)
	truncated := body.Bytes()[:len(body.Bytes())-5]
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testUploadPath, bytes.NewReader(truncated))
	r.Header.Set("Content-Type", ct)

	fatal, err := BindForm(r)

	assert.TrueT(t, fatal)
	assertParseError(t, err, "body", nil)
}

func TestBindForm_singleRequired_present(t *testing.T) {
	r := newMultipartRequest(t, []multipartFile{{testFieldFile, "hello.txt", testValueHello}}, nil)

	var bound *File
	fatal, err := BindForm(r, BindFormFile(testFieldFile, true, func(f multipart.File, h *multipart.FileHeader) error {
		bound = &File{Data: f, Header: h}
		return nil
	}))

	assert.FalseT(t, fatal)
	require.NoError(t, err)
	require.NotNil(t, bound)
	assert.EqualT(t, "hello.txt", bound.Header.Filename)
}

func TestBindForm_singleRequired_missing(t *testing.T) {
	r := newMultipartRequest(t, nil, map[string]string{testFieldDesc: "x"})

	called := false
	fatal, err := BindForm(r, BindFormFile(testFieldFile, true, func(_ multipart.File, _ *multipart.FileHeader) error {
		called = true
		return nil
	}))

	assert.FalseT(t, fatal)
	assert.FalseT(t, called)
	assertCompositeContains(t, err, 1, func(e error) bool {
		var apiErr errors.Error
		if !stderrors.As(e, &apiErr) {
			return false
		}
		return apiErr.Code() == http.StatusBadRequest &&
			strings.Contains(apiErr.Error(), http.ErrMissingFile.Error())
	})
}

func TestBindForm_optional_missing(t *testing.T) {
	r := newMultipartRequest(t, nil, map[string]string{testFieldDesc: "x"})

	called := false
	fatal, err := BindForm(r, BindFormFile(testFieldFile, false, func(_ multipart.File, _ *multipart.FileHeader) error {
		called = true
		return nil
	}))

	assert.FalseT(t, fatal)
	require.NoError(t, err)
	assert.FalseT(t, called, "optional missing file should not invoke binder")
}

func TestBindForm_optional_present(t *testing.T) {
	r := newMultipartRequest(t, []multipartFile{{testFieldFile, "hi.txt", "hi"}}, nil)

	called := false
	fatal, err := BindForm(r, BindFormFile(testFieldFile, false, func(f multipart.File, h *multipart.FileHeader) error {
		called = true
		assert.EqualT(t, "hi.txt", h.Filename)
		got, _ := io.ReadAll(f)
		assert.EqualT(t, "hi", string(got))
		return nil
	}))

	assert.FalseT(t, fatal)
	require.NoError(t, err)
	assert.TrueT(t, called)
}

func TestBindForm_mixed_files_and_values(t *testing.T) {
	r := newMultipartRequest(t,
		[]multipartFile{
			{testFieldFile1, testFileFieldA, "AAA"},
			{testFieldFile2, testFileFieldB, "BBBB"},
		},
		map[string]string{testFieldDesc: "two files", "count": "2"},
	)

	var f1, f2 *File
	fatal, err := BindForm(r,
		BindFormFile(testFieldFile1, true, func(f multipart.File, h *multipart.FileHeader) error {
			f1 = &File{Data: f, Header: h}
			return nil
		}),
		BindFormFile(testFieldFile2, false, func(f multipart.File, h *multipart.FileHeader) error {
			f2 = &File{Data: f, Header: h}
			return nil
		}),
	)

	assert.FalseT(t, fatal)
	require.NoError(t, err)
	require.NotNil(t, f1)
	require.NotNil(t, f2)
	assert.EqualT(t, testFileFieldA, f1.Header.Filename)
	assert.EqualT(t, testFileFieldB, f2.Header.Filename)
	assert.EqualT(t, "two files", r.Form.Get(testFieldDesc))
	assert.EqualT(t, "2", r.Form.Get("count"))
}

func TestBindForm_binderError(t *testing.T) {
	r := newMultipartRequest(t, []multipartFile{{testFieldFile, "f.txt", testFileData}}, nil)
	sentinel := stderrors.New("binder rejected")

	fatal, err := BindForm(r, BindFormFile(testFieldFile, true, func(_ multipart.File, _ *multipart.FileHeader) error {
		return sentinel
	}))

	assert.FalseT(t, fatal)
	assertCompositeContains(t, err, 1, func(e error) bool { return stderrors.Is(e, sentinel) })
}

func TestBindForm_multipleBinderErrors(t *testing.T) {
	r := newMultipartRequest(t,
		[]multipartFile{
			{testFieldFile1, testFileFieldA, "A"},
			{testFieldFile2, testFileFieldB, "B"},
		},
		nil,
	)
	errA := stderrors.New("A failed")
	errB := stderrors.New("B failed")

	fatal, err := BindForm(r,
		BindFormFile(testFieldFile1, true, func(_ multipart.File, _ *multipart.FileHeader) error { return errA }),
		BindFormFile(testFieldFile2, true, func(_ multipart.File, _ *multipart.FileHeader) error { return errB }),
	)

	assert.FalseT(t, fatal)
	assertCompositeContains(t, err, 1, func(e error) bool { return stderrors.Is(e, errA) })
	assertCompositeContains(t, err, 1, func(e error) bool { return stderrors.Is(e, errB) })
}

func TestBindForm_maxFiles_exceeded(t *testing.T) {
	r := newMultipartRequest(t,
		[]multipartFile{
			{testFieldFile1, testFileFieldA, "A"},
			{testFieldFile2, testFileFieldB, "B"},
			{"file3", "c.txt", "C"},
		},
		nil,
	)

	bound := 0
	fatal, err := BindForm(r,
		BindFormMaxFiles(2),
		BindFormFile(testFieldFile1, false, func(_ multipart.File, _ *multipart.FileHeader) error {
			bound++
			return nil
		}),
	)

	assert.TrueT(t, fatal)
	assertParseError(t, err, "body", nil)
	assert.EqualT(t, 0, bound, "no binders should run after maxFiles exceeded")
}

func TestBindForm_maxFilenameLen_exceeded(t *testing.T) {
	longName := strings.Repeat("x", 50) + ".txt"
	r := newMultipartRequest(t,
		[]multipartFile{
			{"big", longName, testFileData},
			{"small", testFileFieldOK, testFileData},
		},
		nil,
	)

	smallBound := false
	fatal, err := BindForm(r,
		BindFormMaxFilenameLen(10),
		BindFormFile("big", true, func(_ multipart.File, _ *multipart.FileHeader) error {
			return nil
		}),
		BindFormFile("small", false, func(_ multipart.File, _ *multipart.FileHeader) error {
			smallBound = true
			return nil
		}),
	)

	assert.FalseT(t, fatal)
	assertCompositeContains(t, err, 1, func(e error) bool {
		var pe *errors.ParseError
		if !stderrors.As(e, &pe) {
			return false
		}
		return pe.Name == "big"
	})
	assert.TrueT(t, smallBound, "small file should still bind after big rejected")
}

func TestBindForm_maxMemory_zero(t *testing.T) {
	r := newMultipartRequest(t, []multipartFile{{testFieldFile, testFileFieldOK, testValueHello}}, nil)

	fatal, err := BindForm(r,
		BindFormMaxParseMemory(0),
		BindFormFile(testFieldFile, true, func(_ multipart.File, _ *multipart.FileHeader) error { return nil }),
	)

	assert.FalseT(t, fatal)
	require.NoError(t, err)
}

func TestBindForm_maxBody_underCap(t *testing.T) {
	r := newMultipartRequest(t, []multipartFile{{testFieldFile, testFileFieldOK, testFileData}}, nil)

	fatal, err := BindForm(r,
		BindFormMaxBody(1<<20),
		BindFormFile(testFieldFile, true, func(_ multipart.File, _ *multipart.FileHeader) error { return nil }),
	)

	assert.FalseT(t, fatal)
	require.NoError(t, err)
}

func TestBindForm_maxBody_overCap(t *testing.T) {
	r := newMultipartRequest(t, []multipartFile{{testFieldFile, testFileFieldOK, strings.Repeat("x", 2048)}}, nil)

	fatal, err := BindForm(r,
		BindFormMaxBody(256),
		BindFormFile(testFieldFile, true, func(_ multipart.File, _ *multipart.FileHeader) error { return nil }),
	)

	assert.TrueT(t, fatal)
	require.Error(t, err)
	var apiErr errors.Error
	require.True(t, stderrors.As(err, &apiErr), "expected errors.Error, got %T", err)
	assert.EqualT(t, int32(http.StatusRequestEntityTooLarge), apiErr.Code())
}

func TestBindForm_maxBody_disabled(t *testing.T) {
	// Body well above DefaultMaxUploadBodySize would be expensive; just
	// confirm a 2 MiB body parses with n=-1 (disabled) when the implicit
	// default would otherwise stay at 32 MiB anyway. The point is that
	// passing -1 doesn't itself break parsing.
	r := newMultipartRequest(t, []multipartFile{{testFieldFile, testFileFieldOK, strings.Repeat("x", 2<<20)}}, nil)

	fatal, err := BindForm(r,
		BindFormMaxBody(-1),
		BindFormFile(testFieldFile, true, func(_ multipart.File, _ *multipart.FileHeader) error { return nil }),
	)

	assert.FalseT(t, fatal)
	require.NoError(t, err)
}

func TestBindForm_idempotent(t *testing.T) {
	r := newMultipartRequest(t, []multipartFile{{testFieldFile, testFileFieldOK, testValueHello}}, map[string]string{testFieldDesc: "x"})

	fatal1, err1 := BindForm(r, BindFormFile(testFieldFile, true, func(_ multipart.File, _ *multipart.FileHeader) error { return nil }))
	require.NoError(t, err1)
	assert.FalseT(t, fatal1)

	// Re-call: stdlib short-circuits because r.MultipartForm != nil.
	fatal2, err2 := BindForm(r, BindFormFile(testFieldFile, true, func(_ multipart.File, _ *multipart.FileHeader) error { return nil }))
	require.NoError(t, err2)
	assert.FalseT(t, fatal2)
	assert.EqualT(t, "x", r.Form.Get(testFieldDesc))
}

// TestValidateFilenameLength covers the exported helper used by both
// BindForm's BindFormFile path and the untyped middleware/parameter.go
// formData path. Security scrub Lens 3 / L3.1.
func TestValidateFilenameLength(t *testing.T) {
	t.Run("within cap returns nil", func(t *testing.T) {
		require.NoError(t, ValidateFilenameLength("avatar", "formData", "ok.txt", 1024))
	})
	t.Run("at cap returns nil", func(t *testing.T) {
		name := strings.Repeat("x", 10)
		require.NoError(t, ValidateFilenameLength("avatar", "formData", name, 10))
	})
	t.Run("over cap returns ParseError", func(t *testing.T) {
		name := strings.Repeat("x", 50)
		err := ValidateFilenameLength("avatar", "formData", name, 10)
		require.Error(t, err)
		var pe *errors.ParseError
		require.True(t, stderrors.As(err, &pe))
		assert.EqualT(t, "avatar", pe.Name)
		assert.EqualT(t, "formData", pe.In)
	})
	t.Run("preview is truncated", func(t *testing.T) {
		name := strings.Repeat("y", 200)
		err := ValidateFilenameLength("avatar", "formData", name, 10)
		require.Error(t, err)
		var pe *errors.ParseError
		require.True(t, stderrors.As(err, &pe))
		// preview must fit filenamePreviewLen (32 bytes).
		assert.LessOrEqual(t, len(pe.Value), filenamePreviewLen)
	})
	t.Run("maxLen<=0 disables the cap", func(t *testing.T) {
		name := strings.Repeat("z", 10000)
		require.NoError(t, ValidateFilenameLength("avatar", "formData", name, 0))
		require.NoError(t, ValidateFilenameLength("avatar", "formData", name, -1))
	})
}
