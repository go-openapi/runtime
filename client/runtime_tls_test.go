package client

import (
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	goruntime "runtime"
	"testing"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeTLSOptions(t *testing.T) {
	fixtures := newTLSFixtures(t)

	t.Run("with TLSAuthConfig configured with files", func(t *testing.T) {
		opts := TLSClientOptions{
			CA:          fixtures.RSA.CAFile,
			Key:         fixtures.RSA.KeyFile,
			Certificate: fixtures.RSA.CertFile,
			ServerName:  fixtures.Subject,
		}

		cfg, err := TLSClientAuth(opts)
		require.NoError(t, err)

		require.NotNil(t, cfg)
		assert.Len(t, cfg.Certificates, 1)
		assert.NotNil(t, cfg.RootCAs)
		assert.Equal(t, fixtures.Subject, cfg.ServerName)
	})

	t.Run("with loaded TLS material", func(t *testing.T) {
		t.Run("with TLSConfig from loaded RSA key/cert pair", func(t *testing.T) {
			opts := TLSClientOptions{
				LoadedKey:         fixtures.RSA.LoadedKey,
				LoadedCertificate: fixtures.RSA.LoadedCert,
			}

			cfg, err := TLSClientAuth(opts)
			require.NoError(t, err)
			require.NotNil(t, cfg)
			assert.Len(t, cfg.Certificates, 1)
		})

		t.Run("with TLSAuthConfig configured with loaded TLS Elliptic Curve key/certificate", func(t *testing.T) {
			opts := TLSClientOptions{
				LoadedKey:         fixtures.ECDSA.LoadedKey,
				LoadedCertificate: fixtures.ECDSA.LoadedCert,
			}

			cfg, err := TLSClientAuth(opts)
			require.NoError(t, err)
			require.NotNil(t, cfg)
			assert.Len(t, cfg.Certificates, 1)
		})

		t.Run("with TLSAuthConfig configured with loaded Certificate Authority", func(t *testing.T) {
			opts := TLSClientOptions{
				LoadedCA: fixtures.RSA.LoadedCA,
			}

			cfg, err := TLSClientAuth(opts)
			require.NoError(t, err)
			require.NotNil(t, cfg)
			assert.NotNil(t, cfg.RootCAs)
		})

		t.Run("with TLSAuthConfig configured with loaded CA pool", func(t *testing.T) {
			pool := x509.NewCertPool()
			pool.AddCert(fixtures.RSA.LoadedCA)

			opts := TLSClientOptions{
				LoadedCAPool: pool,
			}

			cfg, err := TLSClientAuth(opts)
			require.NoError(t, err)
			require.NotNil(t, cfg)
			require.NotNil(t, cfg.RootCAs)
			require.Equal(t, pool, cfg.RootCAs)
		})

		t.Run("with TLSAuthConfig configured with loaded CA and CA pool", func(t *testing.T) {
			pool := systemCAPool(t)
			opts := TLSClientOptions{
				LoadedCAPool: pool,
				LoadedCA:     fixtures.RSA.LoadedCA,
			}

			cfg, err := TLSClientAuth(opts)
			require.NoError(t, err)
			require.NotNil(t, cfg)
			require.NotNil(t, cfg.RootCAs)

			// verify that the CA cert is indeed valid against the configured pool.
			// NOTE: fixtures may be expired certs, but may validate with a fixed date in the past.
			chains, err := fixtures.RSA.LoadedCA.Verify(x509.VerifyOptions{
				Roots:       cfg.RootCAs,
				CurrentTime: time.Date(2017, 1, 1, 1, 1, 1, 1, time.UTC),
			})
			require.NoError(t, err)
			require.NotEmpty(t, chains)
		})

		t.Run("with TLSAuthConfig with VerifyPeer option", func(t *testing.T) {
			verify := func(_ [][]byte, _ [][]*x509.Certificate) error {
				return nil
			}

			opts := TLSClientOptions{
				InsecureSkipVerify:    true,
				VerifyPeerCertificate: verify,
			}

			cfg, err := TLSClientAuth(opts)
			require.NoError(t, err)
			require.NotNil(t, cfg)
			assert.True(t, cfg.InsecureSkipVerify)
			assert.NotNil(t, cfg.VerifyPeerCertificate)
		})
	})
}

func TestRuntimeManualCertificateValidation(t *testing.T) {
	// test manual verification of server certificates
	// against root certificate on client side.
	//
	// The client compares the received cert against the root cert,
	// explicitly omitting DNSName check.
	fixtures := newTLSFixtures(t)
	result := []task{
		{false, "task 1 content", 1},
		{false, "task 2 content", 2},
	}
	host, clean := testTLSServer(t, fixtures, result)
	t.Cleanup(clean)
	var certVerifyCalled bool
	client := testTLSClient(t, fixtures, &certVerifyCalled)
	rt := NewWithClient(host, "/", []string{schemeHTTPS}, client)

	var received []task
	operation := &runtime.ClientOperation{
		ID:          "getTasks",
		Method:      http.MethodGet,
		PathPattern: "/",
		Params: runtime.ClientRequestWriterFunc(func(_ runtime.ClientRequest, _ strfmt.Registry) error {
			return nil
		}),
		Reader: runtime.ClientResponseReaderFunc(func(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
			if response.Code() == http.StatusOK {
				if e := consumer.Consume(response.Body(), &received); e != nil {
					return nil, e
				}
				return result, nil
			}
			return nil, errors.New("generic error")
		}),
	}

	resp, err := rt.Submit(operation)
	require.NoError(t, err)

	require.NotEmpty(t, resp)
	assert.IsType(t, []task{}, resp)

	assert.Truef(t, certVerifyCalled, "the client cert verification has not been called")
	assert.EqualValues(t, result, received)
}

func testTLSServer(t testing.TB, fixtures *tlsFixtures, expectedResult []task) (string, func()) {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Add(runtime.HeaderContentType, runtime.JSONMime)
		rw.WriteHeader(http.StatusOK)
		jsongen := json.NewEncoder(rw)
		require.NoError(t, jsongen.Encode(expectedResult))
	}))

	// create server tls config
	serverCACertPool := x509.NewCertPool()
	serverCACertPool.AddCert(fixtures.Server.LoadedCA)
	// load server certs
	serverCert, err := tls.LoadX509KeyPair(
		fixtures.Server.CertFile,
		fixtures.Server.KeyFile,
	)
	require.NoError(t, err)

	server.TLS = &tls.Config{
		RootCAs:      serverCACertPool,
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{serverCert},
	}
	require.NoError(t, err)

	server.StartTLS()
	testURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	return testURL.Host, server.Close
}

func testTLSClient(t testing.TB, fixtures *tlsFixtures, verifyCalled *bool) *http.Client {
	client, err := TLSClient(TLSClientOptions{
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			*verifyCalled = true

			caCertPool := x509.NewCertPool()
			caCertPool.AddCert(fixtures.RSA.LoadedCA)

			opts := x509.VerifyOptions{
				Roots:       caCertPool,
				CurrentTime: time.Date(2017, time.July, 1, 1, 1, 1, 1, time.UTC),
			}

			cert, e := x509.ParseCertificate(rawCerts[0])
			if e != nil {
				return e
			}

			_, e = cert.Verify(opts)
			return e
		},
	})
	require.NoError(t, err)

	return client
}

type (
	tlsFixtures struct {
		RSA     tlsFixture
		ECDSA   tlsFixture
		Server  tlsFixture
		Subject string
	}

	tlsFixture struct {
		LoadedCA   *x509.Certificate
		LoadedCert *x509.Certificate
		LoadedKey  crypto.PrivateKey

		CAFile   string
		KeyFile  string
		CertFile string
	}
)

// newTLSFixtures loads TLS material for testing
func newTLSFixtures(t testing.TB) *tlsFixtures {
	const subject = "somewhere"

	certFixturesDir := filepath.Join("..", "fixtures", "certs")

	keyFile := filepath.Join(certFixturesDir, "myclient.key")
	keyPem, err := os.ReadFile(keyFile)
	require.NoError(t, err)

	keyDer, _ := pem.Decode(keyPem)
	require.NotNil(t, keyDer)

	key, err := x509.ParsePKCS1PrivateKey(keyDer.Bytes)
	require.NoError(t, err)

	certFile := filepath.Join(certFixturesDir, "myclient.crt")
	certPem, err := os.ReadFile(certFile)
	require.NoError(t, err)

	certDer, _ := pem.Decode(certPem)
	require.NotNil(t, certDer)

	cert, err := x509.ParseCertificate(certDer.Bytes)
	require.NoError(t, err)

	eccKeyFile := filepath.Join(certFixturesDir, "myclient-ecc.key")
	eckeyPem, err := os.ReadFile(eccKeyFile)
	require.NoError(t, err)

	_, remainder := pem.Decode(eckeyPem)
	ecKeyDer, _ := pem.Decode(remainder)
	require.NotNil(t, ecKeyDer)

	ecKey, err := x509.ParseECPrivateKey(ecKeyDer.Bytes)
	require.NoError(t, err)

	eccCertFile := filepath.Join(certFixturesDir, "myclient-ecc.crt")
	ecCertPem, err := os.ReadFile(eccCertFile)
	require.NoError(t, err)

	ecCertDer, _ := pem.Decode(ecCertPem)
	require.NotNil(t, ecCertDer)

	ecCert, err := x509.ParseCertificate(ecCertDer.Bytes)
	require.NoError(t, err)

	caFile := filepath.Join(certFixturesDir, "myCA.crt")
	caPem, err := os.ReadFile(caFile)
	require.NoError(t, err)

	caBlock, _ := pem.Decode(caPem)
	require.NotNil(t, caBlock)

	caCert, err := x509.ParseCertificate(caBlock.Bytes)
	require.NoError(t, err)

	serverKeyFile := filepath.Join(certFixturesDir, "mycert1.key")
	serverKeyPem, err := os.ReadFile(serverKeyFile)
	require.NoError(t, err)

	serverKeyDer, _ := pem.Decode(serverKeyPem)
	require.NotNil(t, serverKeyDer)

	serverKey, err := x509.ParsePKCS1PrivateKey(serverKeyDer.Bytes)
	require.NoError(t, err)

	serverCertFile := filepath.Join(certFixturesDir, "mycert1.crt")
	serverCertPem, err := os.ReadFile(serverCertFile)
	require.NoError(t, err)

	serverCertDer, _ := pem.Decode(serverCertPem)
	require.NotNil(t, serverCertDer)

	serverCert, err := x509.ParseCertificate(serverCertDer.Bytes)
	require.NoError(t, err)

	return &tlsFixtures{
		Subject: subject,
		RSA: tlsFixture{
			CAFile:     caFile,
			KeyFile:    keyFile,
			CertFile:   certFile,
			LoadedCA:   caCert,
			LoadedKey:  key,
			LoadedCert: cert,
		},
		ECDSA: tlsFixture{
			KeyFile:    eccKeyFile,
			CertFile:   eccCertFile,
			LoadedKey:  ecKey,
			LoadedCert: ecCert,
		},
		Server: tlsFixture{
			KeyFile:    serverKeyFile,
			CertFile:   serverCertFile,
			LoadedCA:   caCert,
			LoadedKey:  serverKey,
			LoadedCert: serverCert,
		},
	}
}

func systemCAPool(t testing.TB) *x509.CertPool {
	if goruntime.GOOS == "windows" {
		// Windows doesn't have the system cert pool.
		return x509.NewCertPool()
	}

	pool, err := x509.SystemCertPool()
	require.NoError(t, err)

	return pool
}
