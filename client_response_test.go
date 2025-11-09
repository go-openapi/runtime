// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

type response struct {
}

func (r response) Code() int {
	return 490
}
func (r response) Message() string {
	return "the message"
}
func (r response) GetHeader(_ string) string {
	return "the header"
}
func (r response) GetHeaders(_ string) []string {
	return []string{"the headers", "the headers2"}
}
func (r response) Body() io.ReadCloser {
	return io.NopCloser(bytes.NewBufferString("the content"))
}

func TestResponseReaderFunc(t *testing.T) {
	var actual struct {
		Header, Message, Body string
		Code                  int
	}
	reader := ClientResponseReaderFunc(func(r ClientResponse, _ Consumer) (any, error) {
		b, _ := io.ReadAll(r.Body())
		actual.Body = string(b)
		actual.Code = r.Code()
		actual.Message = r.Message()
		actual.Header = r.GetHeader("blah")
		return actual, nil
	})
	_, _ = reader.ReadResponse(response{}, nil)
	assert.Equal(t, "the content", actual.Body)
	assert.Equal(t, "the message", actual.Message)
	assert.Equal(t, "the header", actual.Header)
	assert.Equal(t, 490, actual.Code)
}

type errResponse struct {
	A int    `json:"a"`
	B string `json:"b"`
}

func TestResponseReaderFuncError(t *testing.T) {
	t.Parallel()

	t.Run("with API error as string", func(t *testing.T) {
		reader := ClientResponseReaderFunc(func(r ClientResponse, _ Consumer) (any, error) {
			_, _ = io.ReadAll(r.Body())

			return nil, NewAPIError("fake", errors.New("writer closed"), 490)
		})

		_, err := reader.ReadResponse(response{}, nil)
		require.Error(t, err)
		require.ErrorContains(t, err, "'writer closed'")
	})

	t.Run("with API error as complex error", func(t *testing.T) {
		reader := ClientResponseReaderFunc(func(r ClientResponse, _ Consumer) (any, error) {
			_, _ = io.ReadAll(r.Body())
			err := &fs.PathError{
				Op:   "write",
				Path: "path/to/fake",
				Err:  fs.ErrClosed,
			}

			return nil, NewAPIError("fake", err, 200)
		})

		_, err := reader.ReadResponse(response{}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "file already closed")
	})

	t.Run("with API error requiring escaping", func(t *testing.T) {
		reader := ClientResponseReaderFunc(func(r ClientResponse, _ Consumer) (any, error) {
			_, _ = io.ReadAll(r.Body())
			return nil, NewAPIError("fake", errors.New(`writer is \"terminated\" and 'closed'`), 490)
		})

		_, err := reader.ReadResponse(response{}, nil)
		require.Error(t, err)
		require.ErrorContains(t, err, `'writer is \\"terminated\\" and \'closed\''`)
	})

	t.Run("with API error as JSON", func(t *testing.T) {
		reader := ClientResponseReaderFunc(func(r ClientResponse, _ Consumer) (any, error) {
			_, _ = io.ReadAll(r.Body())
			obj := &errResponse{ // does not implement error
				A: 555,
				B: "closed",
			}

			return nil, NewAPIError("fake", obj, 200)
		})

		_, err := reader.ReadResponse(response{}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `{"a":555,"b":"closed"}`)
	})
}
