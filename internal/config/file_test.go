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

func TestReadConfig(t *testing.T) {
	t.Run("valid config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yml")

		// Create a valid YAML config file
		yamlContent := `clusters:
  - name: dev
    brokers:
      - localhost:9092
      - localhost:9093
    client_id: ms-dev
  - name: prod
    brokers:
      - kafka1.prod:9092
      - kafka2.prod:9092
    tls:
      enabled: true
      ca_file: /path/to/ca.pem
      cert_file: /path/to/cert.pem
      key_file: /path/to/key.pem
    sasl:
      mechanism: SCRAM-SHA-256
      username: admin
      password: secret
`
		if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Read the config
		cfg, err := ReadConfig(configPath)
		if err != nil {
			t.Fatalf("ReadConfig() error = %v", err)
		}

		// Verify clusters
		if len(cfg.Clusters) != 2 {
			t.Errorf("expected 2 clusters, got %d", len(cfg.Clusters))
		}

		// Verify first cluster
		if cfg.Clusters[0].Name != "dev" {
			t.Errorf("expected cluster name 'dev', got '%s'", cfg.Clusters[0].Name)
		}
		if len(cfg.Clusters[0].Brokers) != 2 {
			t.Errorf("expected 2 brokers, got %d", len(cfg.Clusters[0].Brokers))
		}
		if cfg.Clusters[0].ClientID != "ms-dev" {
			t.Errorf("expected client_id 'ms-dev', got '%s'", cfg.Clusters[0].ClientID)
		}

		// Verify second cluster with TLS and SASL
		if cfg.Clusters[1].Name != "prod" {
			t.Errorf("expected cluster name 'prod', got '%s'", cfg.Clusters[1].Name)
		}
		if cfg.Clusters[1].TLS == nil {
			t.Error("expected TLS config, got nil")
		} else {
			if !cfg.Clusters[1].TLS.Enabled {
				t.Error("expected TLS enabled")
			}
			if cfg.Clusters[1].TLS.CAFile != "/path/to/ca.pem" {
				t.Errorf("expected ca_file '/path/to/ca.pem', got '%s'", cfg.Clusters[1].TLS.CAFile)
			}
		}
		if cfg.Clusters[1].SASL == nil {
			t.Error("expected SASL config, got nil")
		} else {
			if cfg.Clusters[1].SASL.Mechanism != "SCRAM-SHA-256" {
				t.Errorf("expected mechanism 'SCRAM-SHA-256', got '%s'", cfg.Clusters[1].SASL.Mechanism)
			}
			if cfg.Clusters[1].SASL.Username != "admin" {
				t.Errorf("expected username 'admin', got '%s'", cfg.Clusters[1].SASL.Username)
			}
		}
	})

	t.Run("empty config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "empty.yml")

		// Create an empty config file
		yamlContent := `clusters: []`
		if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := ReadConfig(configPath)
		if err != nil {
			t.Fatalf("ReadConfig() error = %v", err)
		}

		if len(cfg.Clusters) != 0 {
			t.Errorf("expected 0 clusters, got %d", len(cfg.Clusters))
		}
	})

	t.Run("config with AWS IAM", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "aws.yml")

		yamlContent := `clusters:
  - name: msk
    brokers:
      - b-1.msk.amazonaws.com:9098
    aws:
      iam: true
      region: us-east-1
      access_key_env: AWS_ACCESS_KEY_ID
      secret_key_env: AWS_SECRET_ACCESS_KEY
`
		if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := ReadConfig(configPath)
		if err != nil {
			t.Fatalf("ReadConfig() error = %v", err)
		}

		if len(cfg.Clusters) != 1 {
			t.Fatalf("expected 1 cluster, got %d", len(cfg.Clusters))
		}

		if cfg.Clusters[0].AWS == nil {
			t.Fatal("expected AWS config, got nil")
		}

		if !cfg.Clusters[0].AWS.IAM {
			t.Error("expected AWS IAM enabled")
		}
		if cfg.Clusters[0].AWS.Region != "us-east-1" {
			t.Errorf("expected region 'us-east-1', got '%s'", cfg.Clusters[0].AWS.Region)
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := ReadConfig("/nonexistent/path/config.yml")
		if err == nil {
			t.Error("expected error for non-existent file, got nil")
		}
	})

	t.Run("invalid YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "invalid.yml")

		// Write invalid YAML
		invalidYAML := `clusters:
  - name: dev
    brokers: [invalid yaml structure
`
		if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := ReadConfig(configPath)
		if err == nil {
			t.Error("expected error for invalid YAML, got nil")
		}
	})

	t.Run("config with options", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "options.yml")

		yamlContent := `clusters:
  - name: dev
    brokers:
      - localhost:9092
    options:
      request.timeout.ms: "30000"
      metadata.max.age.ms: "300000"
`
		if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := ReadConfig(configPath)
		if err != nil {
			t.Fatalf("ReadConfig() error = %v", err)
		}

		if len(cfg.Clusters[0].Options) != 2 {
			t.Errorf("expected 2 options, got %d", len(cfg.Clusters[0].Options))
		}

		if cfg.Clusters[0].Options["request.timeout.ms"] != "30000" {
			t.Errorf("expected option 'request.timeout.ms' = '30000', got '%s'", cfg.Clusters[0].Options["request.timeout.ms"])
		}
	})
}

func TestWriteConfig(t *testing.T) {
	t.Run("write and read back", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yml")

		// Create a config
		originalCfg := FileConfig{
			Clusters: []ClusterConfig{
				{
					Name:     "test",
					Brokers:  []string{"localhost:9092"},
					ClientID: "test-client",
					TLS: &TLSConfig{
						Enabled: true,
						CAFile:  "/path/to/ca.pem",
					},
					SASL: &SASLConfig{
						Mechanism: "PLAIN",
						Username:  "user",
						Password:  "pass",
					},
					Options: map[string]string{
						"key": "value",
					},
				},
			},
		}

		// Write config
		if err := WriteConfig(configPath, originalCfg); err != nil {
			t.Fatalf("WriteConfig() error = %v", err)
		}

		// Read it back
		readCfg, err := ReadConfig(configPath)
		if err != nil {
			t.Fatalf("ReadConfig() error = %v", err)
		}

		// Verify
		if len(readCfg.Clusters) != 1 {
			t.Fatalf("expected 1 cluster, got %d", len(readCfg.Clusters))
		}

		cluster := readCfg.Clusters[0]
		if cluster.Name != "test" {
			t.Errorf("expected name 'test', got '%s'", cluster.Name)
		}
		if cluster.ClientID != "test-client" {
			t.Errorf("expected client_id 'test-client', got '%s'", cluster.ClientID)
		}
		if cluster.TLS == nil || !cluster.TLS.Enabled {
			t.Error("expected TLS enabled")
		}
		if cluster.SASL == nil || cluster.SASL.Mechanism != "PLAIN" {
			t.Error("expected SASL PLAIN mechanism")
		}
		if cluster.Options["key"] != "value" {
			t.Errorf("expected option key='value', got '%s'", cluster.Options["key"])
		}
	})

	t.Run("write to invalid path", func(t *testing.T) {
		cfg := FileConfig{
			Clusters: []ClusterConfig{
				{Name: "test", Brokers: []string{"localhost:9092"}},
			},
		}

		// Try to write to an invalid path (directory doesn't exist)
		err := WriteConfig("/nonexistent/directory/config.yml", cfg)
		if err == nil {
			t.Error("expected error for invalid path, got nil")
		}
	})

	t.Run("write empty config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "empty.yml")

		cfg := FileConfig{
			Clusters: []ClusterConfig{},
		}

		if err := WriteConfig(configPath, cfg); err != nil {
			t.Fatalf("WriteConfig() error = %v", err)
		}

		// Read it back
		readCfg, err := ReadConfig(configPath)
		if err != nil {
			t.Fatalf("ReadConfig() error = %v", err)
		}

		if len(readCfg.Clusters) != 0 {
			t.Errorf("expected 0 clusters, got %d", len(readCfg.Clusters))
		}
	})
}

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
