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

package runtime

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

type eofReader struct{}

func (e *eofReader) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

func closeReader(rdr io.Reader) *closeCounting {
	return &closeCounting{
		rdr: rdr,
	}
}

type closeCounting struct {
	rdr    io.Reader
	closed int
}

func (c *closeCounting) Read(d []byte) (int, error) {
	return c.rdr.Read(d)
}

func (c *closeCounting) Close() error {
	c.closed++
	if cr, ok := c.rdr.(io.ReadCloser); ok {
		return cr.Close()
	}
	return nil
}

type countingBufioReader struct {
	buffereds int
	peeks     int
	reads     int

	br interface {
		Buffered() int
		Peek(int) ([]byte, error)
		Read([]byte) (int, error)
	}
}

func (c *countingBufioReader) Buffered() int {
	c.buffereds++
	return c.br.Buffered()
}

func (c *countingBufioReader) Peek(v int) ([]byte, error) {
	c.peeks++
	return c.br.Peek(v)
}

func (c *countingBufioReader) Read(p []byte) (int, error) {
	c.reads++
	return c.br.Read(p)
}

func TestPeekingReader(t *testing.T) {
	// just passes to original reader when nothing called
	exp1 := []byte("original")
	pr1 := newPeekingReader(closeReader(bytes.NewReader(exp1)))
	b1, err := io.ReadAll(pr1)
	require.NoError(t, err)
	assert.Equal(t, exp1, b1)

	// uses actual when there was some buffering
	exp2 := []byte("actual")
	pr2 := newPeekingReader(closeReader(bytes.NewReader(exp2)))
	peeked, err := pr2.underlying.Peek(1)
	require.NoError(t, err)
	require.Equal(t, "a", string(peeked))
	b2, err := io.ReadAll(pr2)
	require.NoError(t, err)
	assert.Equal(t, string(exp2), string(b2))

	// passes close call through to original reader
	cr := closeReader(closeReader(bytes.NewReader(exp2)))
	pr3 := newPeekingReader(cr)
	require.NoError(t, pr3.Close())
	require.Equal(t, 1, cr.closed)

	// returns false when the stream is empty
	pr4 := newPeekingReader(closeReader(&eofReader{}))
	require.False(t, pr4.HasContent())

	// returns true when the stream has content
	rdr := closeReader(strings.NewReader("hello"))
	pr := newPeekingReader(rdr)
	cbr := &countingBufioReader{
		br: bufio.NewReader(rdr),
	}
	pr.underlying = cbr

	require.True(t, pr.HasContent())
	require.Equal(t, 1, cbr.buffereds)
	require.Equal(t, 1, cbr.peeks)
	require.Equal(t, 0, cbr.reads)
	require.True(t, pr.HasContent())
	require.Equal(t, 2, cbr.buffereds)
	require.Equal(t, 1, cbr.peeks)
	require.Equal(t, 0, cbr.reads)

	b, err := io.ReadAll(pr)
	require.NoError(t, err)
	require.Equal(t, "hello", string(b))
	require.Equal(t, 2, cbr.buffereds)
	require.Equal(t, 1, cbr.peeks)
	require.Equal(t, 2, cbr.reads)
	require.Equal(t, 0, cbr.br.Buffered())

	t.Run("closing a closed peekingReader", func(t *testing.T) {
		const content = "content"
		r := newPeekingReader(io.NopCloser(strings.NewReader(content)))
		require.NoError(t, r.Close())

		require.NotPanics(t, func() {
			err := r.Close()
			require.Error(t, err)
		})
	})

	t.Run("reading from a closed peekingReader", func(t *testing.T) {
		const content = "content"
		r := newPeekingReader(io.NopCloser(strings.NewReader(content)))
		require.NoError(t, r.Close())

		require.NotPanics(t, func() {
			_, err := io.ReadAll(r)
			require.Error(t, err)
			require.ErrorIs(t, err, io.ErrUnexpectedEOF)
		})
	})

	t.Run("reading from a nil peekingReader", func(t *testing.T) {
		var r *peekingReader
		require.NotPanics(t, func() {
			buf := make([]byte, 10)
			_, err := r.Read(buf)
			require.Error(t, err)
			require.ErrorIs(t, err, io.EOF)
		})
	})
}

func TestJSONRequest(t *testing.T) {
	req, err := JSONRequest(http.MethodGet, "/swagger.json", nil)
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, req.Method)
	assert.Equal(t, JSONMime, req.Header.Get(HeaderContentType))
	assert.Equal(t, JSONMime, req.Header.Get(HeaderAccept))

	req, err = JSONRequest(http.MethodGet, "%2", nil)
	require.Error(t, err)
	assert.Nil(t, req)
}

func TestHasBody(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "", nil)
	require.NoError(t, err)
	assert.False(t, HasBody(req))

	req.ContentLength = 123
	assert.True(t, HasBody(req))
}

func TestMethod(t *testing.T) {
	testcase := []struct {
		method      string
		canHaveBody bool
		allowsBody  bool
		isSafe      bool
	}{
		{"put", true, true, false},
		{"post", true, true, false},
		{"patch", true, true, false},
		{"delete", true, true, false},
		{"get", false, true, true},
		{"options", false, true, false},
		{"head", false, false, true},
		{"invalid", false, true, false},
		{"", false, true, false},
	}

	for _, tc := range testcase {
		t.Run(tc.method, func(t *testing.T) {
			assert.Equal(t, tc.canHaveBody, CanHaveBody(tc.method), "CanHaveBody")

			req := http.Request{Method: tc.method}
			assert.Equal(t, tc.allowsBody, AllowsBody(&req), "AllowsBody")
			assert.Equal(t, tc.isSafe, IsSafe(&req), "IsSafe")
		})
	}
}

func TestReadSingle(t *testing.T) {
	values := url.Values(make(map[string][]string))
	values.Add("something", "the thing")
	assert.Equal(t, "the thing", ReadSingleValue(Values(values), "something"))
	assert.Empty(t, ReadSingleValue(Values(values), "notthere"))
}

func TestReadCollection(t *testing.T) {
	values := url.Values(make(map[string][]string))
	values.Add("something", "value1,value2")
	assert.Equal(t, []string{"value1", "value2"}, ReadCollectionValue(Values(values), "something", "csv"))
	assert.Empty(t, ReadCollectionValue(Values(values), "notthere", ""))
}
