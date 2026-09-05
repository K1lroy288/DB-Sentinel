package tls_test

import (
	"crypto/tls"
	"path/filepath"
	"testing"
)

func TestTLSKeyPairLoading(t *testing.T) {
	certPath := filepath.Join("server.crt")
	keyPath := filepath.Join("server.key")

	_, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		t.Fatalf("failed to load x509 key pair: %v", err)
	}
}