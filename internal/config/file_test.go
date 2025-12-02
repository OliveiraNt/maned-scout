package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetAuthType(t *testing.T) {
	tests := []struct {
		name     string
		config   ClusterConfig
		expected string
	}{
		{
			name:     "PLAINTEXT - no auth",
			config:   ClusterConfig{},
			expected: "PLAINTEXT",
		},
		{
			name: "TLS only",
			config: ClusterConfig{
				TLS: &TLSConfig{
					Enabled: true,
					CAFile:  "ca.pem",
				},
			},
			expected: "TLS",
		},
		{
			name: "mTLS - with client certs",
			config: ClusterConfig{
				TLS: &TLSConfig{
					Enabled:  true,
					CAFile:   "ca.pem",
					CertFile: "client.pem",
					KeyFile:  "client-key.pem",
				},
			},
			expected: "mTLS",
		},
		{
			name: "SASL/PLAIN",
			config: ClusterConfig{
				SASL: &SASLConfig{
					Mechanism: "PLAIN",
					Username:  "user",
					Password:  "pass",
				},
			},
			expected: "SASL/PLAIN",
		},
		{
			name: "SASL/SCRAM-SHA-256",
			config: ClusterConfig{
				SASL: &SASLConfig{
					Mechanism: "SCRAM-SHA-256",
					Username:  "user",
					Password:  "pass",
				},
			},
			expected: "SASL/SCRAM-SHA-256",
		},
		{
			name: "SASL/PLAIN + TLS",
			config: ClusterConfig{
				TLS: &TLSConfig{
					Enabled: true,
					CAFile:  "ca.pem",
				},
				SASL: &SASLConfig{
					Mechanism: "PLAIN",
					Username:  "user",
					Password:  "pass",
				},
			},
			expected: "SASL/PLAIN + TLS",
		},
		{
			name: "AWS IAM",
			config: ClusterConfig{
				AWS: &AWSConfig{
					IAM: true,
				},
			},
			expected: "AWS IAM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetAuthType()
			if result != tt.expected {
				t.Errorf("GetAuthType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHasCertificate(t *testing.T) {
	tests := []struct {
		name     string
		config   ClusterConfig
		expected bool
	}{
		{
			name:     "no TLS",
			config:   ClusterConfig{},
			expected: false,
		},
		{
			name: "TLS disabled",
			config: ClusterConfig{
				TLS: &TLSConfig{
					Enabled:  false,
					CertFile: "cert.pem",
				},
			},
			expected: false,
		},
		{
			name: "TLS enabled without cert",
			config: ClusterConfig{
				TLS: &TLSConfig{
					Enabled: true,
				},
			},
			expected: false,
		},
		{
			name: "TLS with cert",
			config: ClusterConfig{
				TLS: &TLSConfig{
					Enabled:  true,
					CertFile: "cert.pem",
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.HasCertificate()
			if result != tt.expected {
				t.Errorf("HasCertificate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetCertificateInfo(t *testing.T) {
	// Create a temporary directory for test certificates
	tmpDir := t.TempDir()

	// Helper to create a test certificate
	createTestCert := func(filename string, notBefore, notAfter time.Time) string {
		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatal(err)
		}

		template := x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject: pkix.Name{
				Organization: []string{"Test"},
			},
			NotBefore:             notBefore,
			NotAfter:              notAfter,
			KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			BasicConstraintsValid: true,
		}

		derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
		if err != nil {
			t.Fatal(err)
		}

		certPath := filepath.Join(tmpDir, filename)
		certOut, err := os.Create(certPath)
		if err != nil {
			t.Fatal(err)
		}
		defer certOut.Close()

		if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
			t.Fatal(err)
		}

		return certPath
	}

	now := time.Now()

	tests := []struct {
		name           string
		certFile       string
		expectedStatus string
	}{
		{
			name:           "valid certificate (90 days)",
			certFile:       createTestCert("valid.pem", now.AddDate(0, 0, -10), now.AddDate(0, 0, 90)),
			expectedStatus: "valid",
		},
		{
			name:           "warning certificate (20 days)",
			certFile:       createTestCert("warning.pem", now.AddDate(0, 0, -10), now.AddDate(0, 0, 20)),
			expectedStatus: "warning",
		},
		{
			name:           "critical certificate (5 days)",
			certFile:       createTestCert("critical.pem", now.AddDate(0, 0, -10), now.AddDate(0, 0, 5)),
			expectedStatus: "critical",
		},
		{
			name:           "expired certificate",
			certFile:       createTestCert("expired.pem", now.AddDate(0, 0, -30), now.AddDate(0, 0, -5)),
			expectedStatus: "expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ClusterConfig{
				TLS: &TLSConfig{
					Enabled:  true,
					CertFile: tt.certFile,
				},
			}

			info, err := cfg.GetCertificateInfo()
			if err != nil {
				t.Fatalf("GetCertificateInfo() error = %v", err)
			}

			if info == nil {
				t.Fatal("Expected certificate info, got nil")
			}

			if info.Status != tt.expectedStatus {
				t.Errorf("GetCertificateInfo().Status = %v, want %v", info.Status, tt.expectedStatus)
			}
		})
	}
}

func TestGetCertificateInfo_NoCertificate(t *testing.T) {
	tests := []struct {
		name   string
		config ClusterConfig
	}{
		{
			name:   "no TLS config",
			config: ClusterConfig{},
		},
		{
			name: "TLS disabled",
			config: ClusterConfig{
				TLS: &TLSConfig{
					Enabled: false,
				},
			},
		},
		{
			name: "no cert file",
			config: ClusterConfig{
				TLS: &TLSConfig{
					Enabled: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := tt.config.GetCertificateInfo()
			if err != nil {
				t.Errorf("GetCertificateInfo() unexpected error = %v", err)
			}
			if info != nil {
				t.Errorf("GetCertificateInfo() = %v, want nil", info)
			}
		})
	}
}
