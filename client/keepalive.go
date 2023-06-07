package client

import (
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
)

// KeepAliveTransport drains the remaining body from a response
// so that go will reuse the TCP connections.
// This is not enabled by default because there are servers where
// the response never gets closed and that would make the code hang forever.
// So instead it's provided as a http client middleware that can be used to override
// any request.
func KeepAliveTransport(rt http.RoundTripper) http.RoundTripper {
	return &keepAliveTransport{wrapped: rt}
}

type keepAliveTransport struct {
	wrapped http.RoundTripper
}

func (k *keepAliveTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := k.wrapped.RoundTrip(r)
	if err != nil {
		return resp, fmt.Errorf("making round trip: %w", err)
	}
	resp.Body = &drainingReadCloser{rdr: resp.Body}
	return resp, nil
}

type drainingReadCloser struct {
	rdr     io.ReadCloser
	seenEOF uint32
}

func (d *drainingReadCloser) Read(p []byte) (n int, err error) {
	n, err = d.rdr.Read(p)
	if err == io.EOF || n == 0 {
		atomic.StoreUint32(&d.seenEOF, 1)
	}
	return
}

func (d *drainingReadCloser) Close() error {
	// drain buffer
	if atomic.LoadUint32(&d.seenEOF) != 1 {
		// If the reader side (a HTTP server) is misbehaving, it still may send
		// some bytes, but the closer ignores them to keep the underling
		// connection open.
		//nolint:errcheck
		io.Copy(io.Discard, d.rdr)
	}
	err := d.rdr.Close()
	if err != nil {
		return fmt.Errorf("closing: %w", err)
	}
	return nil
}
