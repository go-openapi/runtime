// SPDX-FileCopyrightText: Copyright 2015-2026 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-openapi/testify/v2/require"
)

const (
	// X509 material: keys and certificates to test TLS
	myClientKey     = "myclient.key"
	myClientCert    = "myclient.crt"
	myCAKey         = "myCA.key"
	myCACert        = "myCA.crt"
	myServerKey     = "mycert1.key"
	myServerCert    = "mycert1.crt"
	myClientECCKey  = "myclient-ecc.key"
	myClientECCCert = "myclient-ecc.crt"
)

// newTLSFixtures loads TLS material for testing.
func newTLSFixtures(t testing.TB) *tlsFixtures {
	const subject = "somewhere"

	certFixturesDir := t.TempDir()
	require.NoError(t, runGenCerts(t, certFixturesDir))

	keyFile := filepath.Join(certFixturesDir, myClientKey)
	keyPem, err := os.ReadFile(keyFile)
	require.NoError(t, err)
	keyDer, _ := pem.Decode(keyPem)
	require.NotNil(t, keyDer)
	key, err := x509.ParsePKCS1PrivateKey(keyDer.Bytes)
	require.NoError(t, err)

	certFile := filepath.Join(certFixturesDir, myClientCert)
	certPem, err := os.ReadFile(certFile)
	require.NoError(t, err)
	certDer, _ := pem.Decode(certPem)
	require.NotNil(t, certDer)
	cert, err := x509.ParseCertificate(certDer.Bytes)
	require.NoError(t, err)

	eccKeyFile := filepath.Join(certFixturesDir, myClientECCKey)
	eckeyPem, err := os.ReadFile(eccKeyFile)
	require.NoError(t, err)
	eccBlock, remainder := pem.Decode(eckeyPem)
	ecKeyDer, _ := pem.Decode(remainder)
	require.Nil(t, ecKeyDer)
	ecKey, err := x509.ParseECPrivateKey(eccBlock.Bytes)
	require.NoError(t, err)

	eccCertFile := filepath.Join(certFixturesDir, myClientECCCert)
	ecCertPem, err := os.ReadFile(eccCertFile)
	require.NoError(t, err)
	ecCertDer, _ := pem.Decode(ecCertPem)
	require.NotNil(t, ecCertDer)
	ecCert, err := x509.ParseCertificate(ecCertDer.Bytes)
	require.NoError(t, err)
	caFile := filepath.Join(certFixturesDir, myCACert)
	caPem, err := os.ReadFile(caFile)
	require.NoError(t, err)
	caBlock, _ := pem.Decode(caPem)
	require.NotNil(t, caBlock)
	caCert, err := x509.ParseCertificate(caBlock.Bytes)
	require.NoError(t, err)

	serverKeyFile := filepath.Join(certFixturesDir, myServerKey)
	serverKeyPem, err := os.ReadFile(serverKeyFile)
	require.NoError(t, err)
	serverKeyDer, _ := pem.Decode(serverKeyPem)
	require.NotNil(t, serverKeyDer)
	serverKey, err := x509.ParseECPrivateKey(serverKeyDer.Bytes)
	require.NoError(t, err)

	serverCertFile := filepath.Join(certFixturesDir, myServerCert)
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

// runGenCerts generates self-signed TLS certificates for the todo-list-errors example.
//
// It produces:
//   - myCA.key / myCA.crt       — self-signed certificate authority
//   - mycert1.key / mycert1.crt — server certificate (CN=goswagger.local)
//   - myclient.key / myclient.crt — RSA client certificate (CN=localhost)
//   - myclient-ecc.key / myclient-ecc.crt — ECDSA client certificate (CN=localhost)
//
// All ECDSA certificates use ECDSA P-256. All certificates are valid for 10 years.
func runGenCerts(t testing.TB, outDir string) error {
	t.Logf("Generating TLS certificates in %s", outDir)

	// Generate CA
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generating CA key: %w", err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Go Swagger"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("creating CA certificate: %w", err)
	}

	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		return fmt.Errorf("parsing CA certificate: %w", err)
	}

	if err := writeKeyPair(outDir, stem(myCACert), caKey, caCertDER); err != nil {
		return err
	}

	// Generate server certificate
	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generating server key: %w", err)
	}

	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "goswagger.local"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"goswagger.local", "localhost", "www.example.com"},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
	}

	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("creating server certificate: %w", err)
	}

	if err := writeKeyPair(outDir, stem(myServerKey), serverKey, serverCertDER); err != nil {
		return err
	}

	// Generate client certificate

	// RSA client cert
	clientRSAKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generating client RSA key: %w", err)
	}

	clientTemplate := makeCertReqTemplate(3)
	clientRSACertDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caCert, &clientRSAKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("creating client RSA certificate: %w", err)
	}

	if err := writePKCS1KeyPair(outDir, stem(myClientCert), clientRSAKey, clientRSACertDER); err != nil {
		return err
	}

	// ECC client cert
	clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generating client key: %w", err)
	}

	clientTemplate = makeCertReqTemplate(4)
	clientCertDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caCert, &clientKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("creating client ECDSA certificate: %w", err)
	}

	if err := writeKeyPair(outDir, stem(myClientECCCert), clientKey, clientCertDER); err != nil {
		return err
	}

	t.Logf("  %s / %s       — certificate authority", myCAKey, myCACert)
	t.Logf("  %s / %s — server (CN=goswagger.local)", myServerKey, myServerCert)
	t.Logf("  %s / %s — client (RSA, CN=localhost)", myClientKey, myClientCert)
	t.Logf("  %s / %s — client (ECDSA, CN=localhost)", myClientECCKey, myClientECCCert)

	return nil
}

func makeCertReqTemplate(n int64) *x509.Certificate {
	return &x509.Certificate{
		SerialNumber: big.NewInt(n),
		Subject: pkix.Name{
			CommonName:   "localhost",
			Country:      []string{"US"},
			Province:     []string{"California"},
			Locality:     []string{"San Francisco"},
			Organization: []string{"go-swagger"},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
}

func writeKeyPair(dir, name string, key *ecdsa.PrivateKey, certDER []byte) error {
	keyPath := filepath.Join(dir, name+".key")
	certPath := filepath.Join(dir, name+".crt")

	// Write private key
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return fmt.Errorf("marshaling %s key: %w", name, err)
	}

	if err := writePEM(keyPath, "EC PRIVATE KEY", keyDER); err != nil {
		return err
	}

	// Write certificate
	if err := writePEM(certPath, "CERTIFICATE", certDER); err != nil {
		return err
	}

	return nil
}

func writePKCS1KeyPair(dir, name string, key *rsa.PrivateKey, certDER []byte) error {
	keyPath := filepath.Join(dir, name+".key")
	certPath := filepath.Join(dir, name+".crt")

	// Write private key
	keyDER := x509.MarshalPKCS1PrivateKey(key)
	if err := writePEM(keyPath, "EC PRIVATE KEY", keyDER); err != nil {
		return err
	}

	// Write certificate
	if err := writePEM(certPath, "CERTIFICATE", certDER); err != nil {
		return err
	}

	return nil
}

func writePEM(path, blockType string, data []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	defer f.Close()

	return pem.Encode(f, &pem.Block{Type: blockType, Bytes: data})
}

func stem(file string) string {
	s := strings.Split(file, ".")

	return s[0]
}
