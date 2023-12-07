package client

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCountingReader(rdr io.Reader, readOnce bool) *countingReadCloser {
	return &countingReadCloser{
		rdr:      rdr,
		readOnce: readOnce,
	}
}

type countingReadCloser struct {
	rdr         io.Reader
	readOnce    bool
	readCalled  int
	closeCalled int
}

func (c *countingReadCloser) Read(b []byte) (int, error) {
	c.readCalled++
	if c.readCalled > 1 && c.readOnce {
		return 0, io.EOF
	}
	return c.rdr.Read(b)
}

func (c *countingReadCloser) Close() error {
	c.closeCalled++
	return nil
}

func TestDrainingReadCloser(t *testing.T) {
	rdr := newCountingReader(bytes.NewBufferString("There are many things to do"), false)
	prevDisc := io.Discard
	disc := bytes.NewBuffer(nil)
	io.Discard = disc
	defer func() { io.Discard = prevDisc }()

	buf := make([]byte, 5)
	ts := &drainingReadCloser{rdr: rdr}
	_, err := ts.Read(buf)
	require.NoError(t, err)
	require.NoError(t, ts.Close())
	assert.Equal(t, "There", string(buf))
	assert.Equal(t, " are many things to do", disc.String())
	assert.Equal(t, 3, rdr.readCalled)
	assert.Equal(t, 1, rdr.closeCalled)
}

func TestDrainingReadCloser_SeenEOF(t *testing.T) {
	rdr := newCountingReader(bytes.NewBufferString("There are many things to do"), true)
	prevDisc := io.Discard
	disc := bytes.NewBuffer(nil)
	io.Discard = disc
	defer func() { io.Discard = prevDisc }()

	buf := make([]byte, 5)
	ts := &drainingReadCloser{rdr: rdr}
	_, err := ts.Read(buf)
	require.NoError(t, err)
	_, err = ts.Read(nil)
	require.ErrorIs(t, err, io.EOF)
	require.NoError(t, ts.Close())
	assert.Equal(t, "There", string(buf))
	assert.Empty(t, disc.String())
	assert.Equal(t, 2, rdr.readCalled)
	assert.Equal(t, 1, rdr.closeCalled)
}
