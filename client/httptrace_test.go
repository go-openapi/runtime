// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// testOpGetOk is a placeholder operation ID used by tests that
// don't care about the value beyond it being non-empty.
const testOpGetOk = "getOk"

// recordingLogger captures Debugf output for trace assertions.
// Printf is a no-op — Trace only ever calls Debugf.
type recordingLogger struct {
	mu    sync.Mutex
	lines []string
}

func (l *recordingLogger) Printf(string, ...any) {}

func (l *recordingLogger) Debugf(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lines = append(l.lines, fmt.Sprintf(format, args...))
}

func (l *recordingLogger) snapshot() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]string, len(l.lines))
	copy(out, l.lines)
	return out
}

// containsLineWith reports whether any captured line contains
// every needle (substring conjunction). Useful when we care about
// ordering loosely or about presence rather than exact wording.
func containsLineWith(lines []string, needles ...string) bool {
	for _, line := range lines {
		ok := true
		for _, n := range needles {
			if !strings.Contains(line, n) {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

// orderedSubsequence asserts that the given prefixes appear in
// `lines` in the given order (not necessarily contiguous).
func orderedSubsequence(t *testing.T, lines []string, prefixes ...string) {
	t.Helper()
	i := 0
	for _, line := range lines {
		if i >= len(prefixes) {
			return
		}
		if strings.Contains(line, prefixes[i]) {
			i++
		}
	}
	if i < len(prefixes) {
		t.Fatalf("expected ordered subsequence %v, only matched %d. lines:\n%s",
			prefixes, i, strings.Join(lines, "\n"))
	}
}

func TestRuntime_Trace_HappyPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Set(runtime.HeaderContentType, runtime.JSONMime)
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	hu, err := url.Parse(server.URL)
	require.NoError(t, err)

	rec := &recordingLogger{}

	rt := New(hu.Host, "/", []string{schemeHTTP})
	rt.Trace = true
	rt.SetLogger(rec)

	rwrtr := runtime.ClientRequestWriterFunc(func(_ runtime.ClientRequest, _ strfmt.Registry) error {
		return nil
	})

	_, err = rt.Submit(&runtime.ClientOperation{
		ID:          testOpGetOk,
		Method:      http.MethodGet,
		PathPattern: "/",
		Params:      rwrtr,
		Reader: runtime.ClientResponseReaderFunc(func(response runtime.ClientResponse, _ runtime.Consumer) (any, error) {
			if response.Code() != http.StatusOK {
				return nil, errors.New("unexpected status")
			}
			return struct{}{}, nil
		}),
	})
	require.NoError(t, err)

	lines := rec.snapshot()
	require.NotEmpty(t, lines, "expected trace output, got none")

	// Opening line includes method + URL.
	assert.True(t, containsLineWith(lines, "[trace]", "GET", server.URL+"/"),
		"opening line missing method+url; got:\n%s", strings.Join(lines, "\n"))

	// Phase sequence: GetConn → DNSStart (httptest uses 127.0.0.1
	// so DNS may be skipped; don't require it) → ConnectStart →
	// ConnectDone → GotConn → WroteHeaders → WroteRequest →
	// GotFirstResponseByte → PutIdleConn → Summary.
	orderedSubsequence(t, lines,
		"GetConn(",
		"ConnectStart(",
		"ConnectDone(",
		"GotConn(",
		"WroteHeaders",
		"WroteRequest",
		"GotFirstResponseByte",
		"Summary:",
	)

	// Summary line ends with a total= field and reflects status 200.
	var summary string
	for _, line := range lines {
		if strings.Contains(line, "Summary:") {
			summary = line
		}
	}
	assert.Contains(t, summary, "200")
	assert.Contains(t, summary, "total=")
}

func TestRuntime_Trace_DisabledByDefault(t *testing.T) {
	// Confirms r.Trace defaults to false even when SWAGGER_DEBUG /
	// DEBUG would have set r.Debug = true. This is the env-var
	// decoupling contract.
	t.Setenv("SWAGGER_DEBUG", "1")
	rt := New("example.com", "/", []string{schemeHTTPS})
	assert.False(t, rt.Trace, "Trace must default to false regardless of SWAGGER_DEBUG")
	// r.Debug remains coupled for now (v2 removal); confirm it's
	// the only one affected.
	assert.True(t, rt.Debug, "Debug seed from SWAGGER_DEBUG still in effect (v1 behaviour)")
}

// TestRuntime_Trace_BodyChunkReceived exercises the response-side
// body wrapper: a server returns a payload large enough to force
// multiple Read calls by the consumer side, and we assert that
// each Read shows up as a BodyChunkReceived event.
func TestRuntime_Trace_BodyChunkReceived(t *testing.T) {
	// 64 KiB payload, read 4 KiB at a time → at least a few
	// BodyChunkReceived events.
	const (
		payloadSize = 64 * 1024
		readSize    = 4 * 1024
	)
	payload := bytes.Repeat([]byte("x"), payloadSize)

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Set(runtime.HeaderContentType, runtime.DefaultMime)
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write(payload)
	}))
	defer server.Close()

	hu, err := url.Parse(server.URL)
	require.NoError(t, err)

	rec := &recordingLogger{}
	rt := New(hu.Host, "/", []string{schemeHTTP})
	rt.Trace = true
	rt.SetLogger(rec)

	rwrtr := runtime.ClientRequestWriterFunc(func(_ runtime.ClientRequest, _ strfmt.Registry) error {
		return nil
	})

	_, err = rt.Submit(&runtime.ClientOperation{
		ID:          "getBlob",
		Method:      http.MethodGet,
		PathPattern: "/",
		Params:      rwrtr,
		Reader: runtime.ClientResponseReaderFunc(func(response runtime.ClientResponse, _ runtime.Consumer) (any, error) {
			// Drain the body in fixed-size chunks so each Read on
			// the wrapped body produces a BodyChunkReceived event.
			buf := make([]byte, readSize)
			var total int
			for {
				n, rerr := response.Body().Read(buf)
				total += n
				if rerr == io.EOF {
					break
				}
				if rerr != nil {
					return nil, rerr
				}
			}
			require.Equal(t, payloadSize, total)
			return struct{}{}, nil
		}),
	})
	require.NoError(t, err)

	lines := rec.snapshot()
	// At least one BodyChunkReceived event should fire. Exact
	// count depends on the Transport's internal buffering.
	count := 0
	for _, line := range lines {
		if strings.Contains(line, "BodyChunkReceived(") {
			count++
		}
	}
	assert.Positive(t, count, "expected at least one BodyChunkReceived; lines:\n%s", strings.Join(lines, "\n"))

	// Subsequent events on the same body should carry a dt= field.
	if count > 1 {
		assert.True(t, containsLineWith(lines, "BodyChunkReceived(", "dt="),
			"expected dt= on a subsequent BodyChunkReceived; lines:\n%s", strings.Join(lines, "\n"))
	}
}

// TestRuntime_Trace_BodyChunkSent exercises the request-side body
// wrapper: a POST with a streaming body should produce at least
// one BodyChunkSent event before WroteRequest.
func TestRuntime_Trace_BodyChunkSent(t *testing.T) {
	// Use a body big enough that Transport actually reads from it.
	const payloadSize = 8 * 1024
	payload := bytes.Repeat([]byte("y"), payloadSize)

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, _ = io.Copy(io.Discard, req.Body)
		rw.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	hu, err := url.Parse(server.URL)
	require.NoError(t, err)

	rec := &recordingLogger{}
	rt := New(hu.Host, "/", []string{schemeHTTP})
	rt.Trace = true
	rt.SetLogger(rec)

	rwrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		return req.SetBodyParam(bytes.NewReader(payload))
	})

	_, err = rt.Submit(&runtime.ClientOperation{
		ID:                 "postBlob",
		Method:             http.MethodPost,
		PathPattern:        "/",
		Params:             rwrtr,
		ProducesMediaTypes: []string{runtime.DefaultMime},
		ConsumesMediaTypes: []string{runtime.DefaultMime},
		Reader: runtime.ClientResponseReaderFunc(func(_ runtime.ClientResponse, _ runtime.Consumer) (any, error) {
			return struct{}{}, nil
		}),
	})
	require.NoError(t, err)

	lines := rec.snapshot()
	sent := 0
	for _, line := range lines {
		if strings.Contains(line, "BodyChunkSent(") {
			sent++
		}
	}
	assert.Positive(t, sent,
		"expected at least one BodyChunkSent on a POST with a non-empty body; lines:\n%s",
		strings.Join(lines, "\n"))

	// BodyChunkSent events must precede WroteRequest in the timeline.
	orderedSubsequence(t, lines, "BodyChunkSent(", "WroteRequest")
}

// TestRuntime_Trace_StaleIdleAnnotation forges a GotConn event
// reporting a long-idle reuse so the HEADS-UP annotation fires
// without depending on real time-passes. We invoke the trace
// session directly because reproducing a 30s+ idle conn through
// the real Transport in a unit test would be both slow and flaky.
func TestRuntime_Trace_StaleIdleAnnotation(t *testing.T) {
	rec := &recordingLogger{}
	sess := newTraceSession(rec, http.MethodGet, "http://example.com/api", nil)

	sess.onGotConn(httptrace.GotConnInfo{
		Reused:   true,
		WasIdle:  true,
		IdleTime: 47 * time.Second,
	})

	lines := rec.snapshot()
	assert.True(t, containsLineWith(lines, "GotConn(reused=true", "idle=true", "idle-time=47s"),
		"GotConn line missing or malformed; got:\n%s", strings.Join(lines, "\n"))
	assert.True(t, containsLineWith(lines, "HEADS-UP", "reused idle connection"),
		"HEADS-UP annotation missing; got:\n%s", strings.Join(lines, "\n"))
	assert.True(t, containsLineWith(lines, "NAT may have dropped"),
		"HEADS-UP body missing NAT pointer; got:\n%s", strings.Join(lines, "\n"))
}

// TestRuntime_Trace_StaleIdleFailureSummary verifies that a
// round-trip error on a stale-idle reused conn triggers the
// issue-#336 tail block in the summary.
func TestRuntime_Trace_StaleIdleFailureSummary(t *testing.T) {
	rec := &recordingLogger{}
	sess := newTraceSession(rec, http.MethodGet, "http://example.com/api", nil)

	sess.onGotConn(httptrace.GotConnInfo{
		Reused:   true,
		WasIdle:  true,
		IdleTime: 90 * time.Second,
	})
	sess.onRoundTripError(io.EOF)
	sess.finish()

	lines := rec.snapshot()
	assert.True(t, containsLineWith(lines, "Summary:", "FAILED", "EOF"),
		"summary line missing FAILED/EOF; got:\n%s", strings.Join(lines, "\n"))
	assert.True(t, containsLineWith(lines, "Silently closed"),
		"issue-#336 tail annotation missing; got:\n%s", strings.Join(lines, "\n"))
	assert.True(t, containsLineWith(lines, "IdleConnTimeout"),
		"tail annotation missing IdleConnTimeout pointer; got:\n%s", strings.Join(lines, "\n"))
}

// TestRuntime_Trace_FreshConnNoAnnotation guards against false
// positives: a freshly-dialed (Reused=false) conn must never
// trigger the HEADS-UP / issue-#336 blocks.
func TestRuntime_Trace_FreshConnNoAnnotation(t *testing.T) {
	rec := &recordingLogger{}
	sess := newTraceSession(rec, http.MethodGet, "http://example.com/api", nil)

	sess.onGotConn(httptrace.GotConnInfo{Reused: false})
	sess.onRoundTripError(io.EOF)
	sess.finish()

	lines := rec.snapshot()
	assert.False(t, containsLineWith(lines, "HEADS-UP"),
		"HEADS-UP should NOT fire on a fresh conn; got:\n%s", strings.Join(lines, "\n"))
	assert.False(t, containsLineWith(lines, "issue-#336"),
		"issue-#336 tail should NOT fire on a fresh conn; got:\n%s", strings.Join(lines, "\n"))
}

// TestRuntime_Trace_ShortIdleNoAnnotation guards the threshold:
// an idle-time below [staleIdleThreshold] must not trigger the
// HEADS-UP block.
func TestRuntime_Trace_ShortIdleNoAnnotation(t *testing.T) {
	rec := &recordingLogger{}
	sess := newTraceSession(rec, http.MethodGet, "http://example.com/api", nil)

	sess.onGotConn(httptrace.GotConnInfo{
		Reused:   true,
		WasIdle:  true,
		IdleTime: 5 * time.Second,
	})

	lines := rec.snapshot()
	assert.False(t, containsLineWith(lines, "HEADS-UP"),
		"HEADS-UP should NOT fire below the threshold; got:\n%s", strings.Join(lines, "\n"))
}

// staleConn implements net.Conn and returns io.EOF on Read after
// a fixed number of writes succeed. Used to simulate a server (or
// NAT) silently closing a conn while it sat in the idle pool.
type staleConn struct {
	mu     sync.Mutex
	closed bool
}

func (c *staleConn) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

func (c *staleConn) Write(p []byte) (int, error) {
	// Pretend the write succeeded so the Transport gets past
	// WroteHeaders/WroteRequest before noticing the conn is dead.
	return len(p), nil
}

func (c *staleConn) Close() error {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
	return nil
}

func (*staleConn) LocalAddr() net.Addr              { return &net.TCPAddr{} }
func (*staleConn) RemoteAddr() net.Addr             { return &net.TCPAddr{} }
func (*staleConn) SetDeadline(time.Time) error      { return nil }
func (*staleConn) SetReadDeadline(time.Time) error  { return nil }
func (*staleConn) SetWriteDeadline(time.Time) error { return nil }

// TestRuntime_Trace_StaleConnRealRoundTrip exercises the full
// SubmitContext path with a Transport whose Dial returns a conn
// that EOFs on read. The round-trip fails; trace output should
// carry a FAILED summary line — but the annotation block does NOT
// fire because the conn is fresh (Reused=false, no idle history).
// This is the boundary case: same symptom, but the diagnostic
// only fires when the data on the GotConn event justifies it.
func TestRuntime_Trace_StaleConnRealRoundTrip(t *testing.T) {
	transport := &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return &staleConn{}, nil
		},
		DisableKeepAlives: true,
	}

	rec := &recordingLogger{}
	rt := New("example.com", "/", []string{schemeHTTP})
	rt.Transport = transport
	rt.Trace = true
	rt.SetLogger(rec)

	rwrtr := runtime.ClientRequestWriterFunc(func(_ runtime.ClientRequest, _ strfmt.Registry) error {
		return nil
	})

	_, err := rt.Submit(&runtime.ClientOperation{
		ID:          testOpGetOk,
		Method:      http.MethodGet,
		PathPattern: "/",
		Params:      rwrtr,
		Reader: runtime.ClientResponseReaderFunc(func(_ runtime.ClientResponse, _ runtime.Consumer) (any, error) {
			return struct{}{}, nil
		}),
	})
	require.Error(t, err)

	lines := rec.snapshot()
	assert.True(t, containsLineWith(lines, "Summary:", "FAILED"),
		"expected FAILED summary; got:\n%s", strings.Join(lines, "\n"))
	// Fresh conn → no HEADS-UP / issue-#336 annotation. This is
	// the correct behaviour: the diagnostic only fires when the
	// connection's reuse history points the finger.
	assert.False(t, containsLineWith(lines, "issue-#336"),
		"issue-#336 must NOT fire on a fresh conn even if it EOFs; got:\n%s",
		strings.Join(lines, "\n"))
}

// ===============================================================
// TLS diagnostic mode
// ===============================================================

// generateTestCert builds a single self-signed ECDSA leaf for use
// in TLS tests. notBefore/notAfter let the caller forge expiry;
// dnsNames is the SAN list embedded in the cert.
func generateTestCert(t *testing.T, notBefore, notAfter time.Time, dnsNames ...string) (tls.Certificate, *x509.Certificate) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "trace-test"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)

	leaf, err := x509.ParseCertificate(der)
	require.NoError(t, err)

	keyBytes, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	require.NoError(t, err)
	return cert, leaf
}

// runTLSFailureSubmit drives a Submit against a TLS server whose
// cert configuration is set by `serverCert`, with a client
// configured via `clientTLS`. Returns the trace lines captured.
func runTLSFailureSubmit(t *testing.T, serverCert tls.Certificate, clientTLS *tls.Config) []string {
	t.Helper()

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}))
	server.TLS = &tls.Config{Certificates: []tls.Certificate{serverCert}}
	server.StartTLS()
	defer server.Close()

	hu, err := url.Parse(server.URL)
	require.NoError(t, err)

	rec := &recordingLogger{}

	rt := New(hu.Host, "/", []string{schemeHTTPS})
	rt.Transport = &http.Transport{TLSClientConfig: clientTLS}
	rt.Trace = true
	rt.SetLogger(rec)

	rwrtr := runtime.ClientRequestWriterFunc(func(_ runtime.ClientRequest, _ strfmt.Registry) error {
		return nil
	})

	_, _ = rt.Submit(&runtime.ClientOperation{
		ID:          testOpGetOk,
		Method:      http.MethodGet,
		PathPattern: "/",
		Params:      rwrtr,
		Reader: runtime.ClientResponseReaderFunc(func(_ runtime.ClientResponse, _ runtime.Consumer) (any, error) {
			return struct{}{}, nil
		}),
	})

	return rec.snapshot()
}

// TestRuntime_Trace_TLS_UnknownAuthority: the default httptest TLS
// server uses a self-signed cert that's not in the system trust
// store, so a vanilla client gets x509.UnknownAuthorityError. The
// diagnostic block should classify it as cert-chain / unknown-CA.
func TestRuntime_Trace_TLS_UnknownAuthority(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	hu, err := url.Parse(server.URL)
	require.NoError(t, err)

	rec := &recordingLogger{}
	rt := New(hu.Host, "/", []string{schemeHTTPS})
	rt.Trace = true
	rt.SetLogger(rec)

	rwrtr := runtime.ClientRequestWriterFunc(func(_ runtime.ClientRequest, _ strfmt.Registry) error {
		return nil
	})

	_, err = rt.Submit(&runtime.ClientOperation{
		ID:          testOpGetOk,
		Method:      http.MethodGet,
		PathPattern: "/",
		Params:      rwrtr,
		Reader: runtime.ClientResponseReaderFunc(func(_ runtime.ClientResponse, _ runtime.Consumer) (any, error) {
			return struct{}{}, nil
		}),
	})
	require.Error(t, err)

	lines := rec.snapshot()
	assert.True(t, containsLineWith(lines, "TLS DIAGNOSTIC"),
		"missing TLS diagnostic header; got:\n%s", strings.Join(lines, "\n"))
	assert.True(t, containsLineWith(lines, "axis: cert-chain"),
		"missing cert-chain axis tag; got:\n%s", strings.Join(lines, "\n"))
	assert.True(t, containsLineWith(lines, "unknown-CA"),
		"missing unknown-CA reason; got:\n%s", strings.Join(lines, "\n"))
	assert.True(t, containsLineWith(lines, "SystemCertPool"),
		"missing trust-store note; got:\n%s", strings.Join(lines, "\n"))
}

// TestRuntime_Trace_TLS_ExpiredCert forges a cert with NotAfter in
// the past; the diagnostic should report reason=expired plus the
// expiry timestamps.
func TestRuntime_Trace_TLS_ExpiredCert(t *testing.T) {
	now := time.Now()
	cert, _ := generateTestCert(t,
		now.Add(-48*time.Hour),
		now.Add(-1*time.Hour),
		"127.0.0.1",
	)

	lines := runTLSFailureSubmit(t, cert, nil)
	assert.True(t, containsLineWith(lines, "TLS DIAGNOSTIC"),
		"missing TLS diagnostic header; got:\n%s", strings.Join(lines, "\n"))
	assert.True(t, containsLineWith(lines, "axis: cert-chain"),
		"missing cert-chain axis tag; got:\n%s", strings.Join(lines, "\n"))
	// On modern Go, an expired self-signed cert is reported as
	// either x509.Expired or, when chain-building gives up first,
	// UnknownAuthorityError. Accept either — both come from the
	// same root cause.
	assert.True(t,
		containsLineWith(lines, "reason:", "expired") || containsLineWith(lines, "unknown-CA"),
		"missing expired/unknown-CA reason; got:\n%s", strings.Join(lines, "\n"))
}

// TestRuntime_Trace_TLS_HostnameMismatch: dial 127.0.0.1 but serve
// a cert whose SANs only cover example.com. The handshake itself
// succeeds; verification on the client side then trips
// x509.HostnameError. The trust setup uses the leaf as its own
// RootCA so the failure mode is hostname, not unknown-CA.
func TestRuntime_Trace_TLS_HostnameMismatch(t *testing.T) {
	now := time.Now()
	cert, leaf := generateTestCert(t,
		now.Add(-1*time.Hour),
		now.Add(24*time.Hour),
		"example.com",
	)

	pool := x509.NewCertPool()
	pool.AddCert(leaf)
	clientTLS := &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    pool,
		ServerName: "wronghost", // forces hostname check at "wronghost"
	}

	lines := runTLSFailureSubmit(t, cert, clientTLS)
	assert.True(t, containsLineWith(lines, "TLS DIAGNOSTIC"),
		"missing TLS diagnostic header; got:\n%s", strings.Join(lines, "\n"))
	assert.True(t, containsLineWith(lines, "axis: cert-chain"),
		"missing cert-chain axis tag; got:\n%s", strings.Join(lines, "\n"))
	assert.True(t, containsLineWith(lines, "hostname mismatch"),
		"missing hostname-mismatch reason; got:\n%s", strings.Join(lines, "\n"))
}

// TestRuntime_Trace_TLS_ProtocolVersionMismatch: client pins
// MinVersion=TLS 1.3, server caps at MaxVersion=TLS 1.2. The
// negotiation fails with a protocol-version alert / error.
func TestRuntime_Trace_TLS_ProtocolVersionMismatch(t *testing.T) {
	now := time.Now()
	cert, leaf := generateTestCert(t,
		now.Add(-1*time.Hour),
		now.Add(24*time.Hour),
		"127.0.0.1",
	)

	pool := x509.NewCertPool()
	pool.AddCert(leaf)

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}))
	server.TLS = &tls.Config{ //nolint:gosec // Deliberately caps at TLS 1.2 to force a protocol-version mismatch with the TLS-1.3-only client.
		Certificates: []tls.Certificate{cert},
		MaxVersion:   tls.VersionTLS12,
	}
	server.StartTLS()
	defer server.Close()

	hu, err := url.Parse(server.URL)
	require.NoError(t, err)

	rec := &recordingLogger{}
	rt := New(hu.Host, "/", []string{schemeHTTPS})
	rt.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS13,
			RootCAs:    pool,
		},
	}
	rt.Trace = true
	rt.SetLogger(rec)

	rwrtr := runtime.ClientRequestWriterFunc(func(_ runtime.ClientRequest, _ strfmt.Registry) error {
		return nil
	})

	_, err = rt.Submit(&runtime.ClientOperation{
		ID:          testOpGetOk,
		Method:      http.MethodGet,
		PathPattern: "/",
		Params:      rwrtr,
		Reader: runtime.ClientResponseReaderFunc(func(_ runtime.ClientResponse, _ runtime.Consumer) (any, error) {
			return struct{}{}, nil
		}),
	})
	require.Error(t, err)

	lines := rec.snapshot()
	assert.True(t, containsLineWith(lines, "TLS DIAGNOSTIC"),
		"missing TLS diagnostic header; got:\n%s", strings.Join(lines, "\n"))
	assert.True(t, containsLineWith(lines, "axis: protocol-version"),
		"expected protocol-version axis; got:\n%s", strings.Join(lines, "\n"))
	assert.True(t, containsLineWith(lines, "TLS 1.3"),
		"expected client-offered TLS 1.3; got:\n%s", strings.Join(lines, "\n"))
}

// TestRuntime_Trace_TLS_HappyPath verifies the happy-path summary
// (negotiated version, cipher, server, expiry) on a successful
// TLS handshake. No diagnostic block should appear.
func TestRuntime_Trace_TLS_HappyPath(t *testing.T) {
	now := time.Now()
	cert, leaf := generateTestCert(t,
		now.Add(-1*time.Hour),
		now.Add(24*time.Hour),
		"127.0.0.1",
	)

	pool := x509.NewCertPool()
	pool.AddCert(leaf)
	clientTLS := &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    pool,
	}

	lines := runTLSFailureSubmit(t, cert, clientTLS)

	// Despite the helper name, this case is the happy path — no
	// TLS diagnostic block, but TLSHandshakeDone reports the
	// negotiated parameters.
	assert.False(t, containsLineWith(lines, "TLS DIAGNOSTIC"),
		"happy path must not emit a TLS diagnostic block; got:\n%s",
		strings.Join(lines, "\n"))
	assert.True(t, containsLineWith(lines, "TLSHandshakeDone(", "cipher=", "expires="),
		"happy path TLSHandshakeDone should report cipher and expiry; got:\n%s",
		strings.Join(lines, "\n"))
}

func TestRuntime_Trace_OffEmitsNothing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Set(runtime.HeaderContentType, runtime.JSONMime)
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	hu, err := url.Parse(server.URL)
	require.NoError(t, err)

	rec := &recordingLogger{}

	rt := New(hu.Host, "/", []string{schemeHTTP})
	// rt.Trace stays false; rt.Debug also false → no output expected.
	rt.Debug = false
	rt.SetLogger(rec)

	rwrtr := runtime.ClientRequestWriterFunc(func(_ runtime.ClientRequest, _ strfmt.Registry) error {
		return nil
	})

	_, err = rt.Submit(&runtime.ClientOperation{
		ID:          testOpGetOk,
		Method:      http.MethodGet,
		PathPattern: "/",
		Params:      rwrtr,
		Reader: runtime.ClientResponseReaderFunc(func(_ runtime.ClientResponse, _ runtime.Consumer) (any, error) {
			return struct{}{}, nil
		}),
	})
	require.NoError(t, err)

	assert.Empty(t, rec.snapshot(), "no trace output expected when Trace=false")
}
