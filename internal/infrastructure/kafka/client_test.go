package kafka

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

	"github.com/OliveiraNt/kdash/internal/config"
	"github.com/OliveiraNt/kdash/internal/domain"
)

func TestNewClient(t *testing.T) {
	t.Run("basic client creation", func(t *testing.T) {
		cfg := config.ClusterConfig{
			Name:     "test",
			Brokers:  []string{"localhost:9092"},
			ClientID: "test-client",
		}

		client, err := NewClient(cfg)
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		if client == nil {
			t.Fatal("expected non-nil client")
		}
		defer client.Close()

		if client.config.Name != "test" {
			t.Errorf("expected name 'test', got '%s'", client.config.Name)
		}
		if client.config.ClientID != "test-client" {
			t.Errorf("expected client_id 'test-client', got '%s'", client.config.ClientID)
		}
	})

	t.Run("client with empty brokers", func(t *testing.T) {
		cfg := config.ClusterConfig{
			Name:    "test",
			Brokers: []string{},
		}

		client, err := NewClient(cfg)
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		if client == nil {
			t.Fatal("expected non-nil client")
		}
		defer client.Close()
	})

	t.Run("client without client_id", func(t *testing.T) {
		cfg := config.ClusterConfig{
			Name:    "test",
			Brokers: []string{"localhost:9092"},
		}

		client, err := NewClient(cfg)
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		if client == nil {
			t.Fatal("expected non-nil client")
		}
		defer client.Close()
	})
}

func TestNewClientWithTLS(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate test CA certificate
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test CA"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}

	caFile := filepath.Join(tmpDir, "ca.pem")
	caOut, err := os.Create(caFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := pem.Encode(caOut, &pem.Block{Type: "CERTIFICATE", Bytes: caCertDER}); err != nil {
		t.Fatal(err)
	}
	caOut.Close()

	t.Run("TLS with CA only", func(t *testing.T) {
		cfg := config.ClusterConfig{
			Name:    "test-tls",
			Brokers: []string{"localhost:9093"},
			TLS: &config.TLSConfig{
				Enabled: true,
				CAFile:  caFile,
			},
		}

		client, err := NewClient(cfg)
		if err != nil {
			t.Fatalf("NewClient() with TLS error = %v", err)
		}
		if client == nil {
			t.Fatal("expected non-nil client")
		}
		defer client.Close()
	})

	t.Run("TLS with insecure skip verify", func(t *testing.T) {
		cfg := config.ClusterConfig{
			Name:    "test-tls",
			Brokers: []string{"localhost:9093"},
			TLS: &config.TLSConfig{
				Enabled:            true,
				InsecureSkipVerify: true,
			},
		}

		client, err := NewClient(cfg)
		if err != nil {
			t.Fatalf("NewClient() with insecure TLS error = %v", err)
		}
		if client == nil {
			t.Fatal("expected non-nil client")
		}
		defer client.Close()
	})

	t.Run("TLS with invalid CA file", func(t *testing.T) {
		cfg := config.ClusterConfig{
			Name:    "test-tls",
			Brokers: []string{"localhost:9093"},
			TLS: &config.TLSConfig{
				Enabled: true,
				CAFile:  "/nonexistent/ca.pem",
			},
		}

		_, err := NewClient(cfg)
		if err == nil {
			t.Error("expected error for invalid CA file, got nil")
		}
	})
}

func TestNewClientWithSASL(t *testing.T) {
	t.Run("SASL PLAIN", func(t *testing.T) {
		cfg := config.ClusterConfig{
			Name:    "test-sasl",
			Brokers: []string{"localhost:9092"},
			SASL: &config.SASLConfig{
				Mechanism: "PLAIN",
				Username:  "admin",
				Password:  "secret",
			},
		}

		client, err := NewClient(cfg)
		if err != nil {
			t.Fatalf("NewClient() with SASL PLAIN error = %v", err)
		}
		if client == nil {
			t.Fatal("expected non-nil client")
		}
		defer client.Close()
	})

	t.Run("SASL SCRAM-SHA-256", func(t *testing.T) {
		cfg := config.ClusterConfig{
			Name:    "test-sasl",
			Brokers: []string{"localhost:9092"},
			SASL: &config.SASLConfig{
				Mechanism: "SCRAM-SHA-256",
				Username:  "admin",
				Password:  "secret",
			},
		}

		client, err := NewClient(cfg)
		if err != nil {
			t.Fatalf("NewClient() with SCRAM-SHA-256 error = %v", err)
		}
		if client == nil {
			t.Fatal("expected non-nil client")
		}
		defer client.Close()
	})

	t.Run("SASL SCRAM-SHA-512", func(t *testing.T) {
		cfg := config.ClusterConfig{
			Name:    "test-sasl",
			Brokers: []string{"localhost:9092"},
			SASL: &config.SASLConfig{
				Mechanism: "SCRAM-SHA-512",
				Username:  "admin",
				Password:  "secret",
			},
		}

		client, err := NewClient(cfg)
		if err != nil {
			t.Fatalf("NewClient() with SCRAM-SHA-512 error = %v", err)
		}
		if client == nil {
			t.Fatal("expected non-nil client")
		}
		defer client.Close()
	})

	t.Run("SASL with env variables", func(t *testing.T) {
		os.Setenv("TEST_USERNAME", "envuser")
		os.Setenv("TEST_PASSWORD", "envpass")
		defer func() {
			os.Unsetenv("TEST_USERNAME")
			os.Unsetenv("TEST_PASSWORD")
		}()

		cfg := config.ClusterConfig{
			Name:    "test-sasl",
			Brokers: []string{"localhost:9092"},
			SASL: &config.SASLConfig{
				Mechanism:   "PLAIN",
				UsernameEnv: "TEST_USERNAME",
				PasswordEnv: "TEST_PASSWORD",
			},
		}

		client, err := NewClient(cfg)
		if err != nil {
			t.Fatalf("NewClient() with SASL env error = %v", err)
		}
		if client == nil {
			t.Fatal("expected non-nil client")
		}
		defer client.Close()
	})

	t.Run("SASL with unknown mechanism", func(t *testing.T) {
		cfg := config.ClusterConfig{
			Name:    "test-sasl",
			Brokers: []string{"localhost:9092"},
			SASL: &config.SASLConfig{
				Mechanism: "UNKNOWN",
				Username:  "admin",
				Password:  "secret",
			},
		}

		client, err := NewClient(cfg)
		if err != nil {
			t.Fatalf("NewClient() with unknown SASL mechanism error = %v", err)
		}
		if client == nil {
			t.Fatal("expected non-nil client")
		}
		defer client.Close()
	})
}

func TestNewClientWithAWS(t *testing.T) {
	t.Run("AWS IAM with env variables", func(t *testing.T) {
		os.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
		os.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
		defer func() {
			os.Unsetenv("AWS_ACCESS_KEY_ID")
			os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		}()

		cfg := config.ClusterConfig{
			Name:    "test-aws",
			Brokers: []string{"b-1.msk.amazonaws.com:9098"},
			AWS: &config.AWSConfig{
				IAM:    true,
				Region: "us-east-1",
			},
		}

		client, err := NewClient(cfg)
		if err != nil {
			t.Fatalf("NewClient() with AWS IAM error = %v", err)
		}
		if client == nil {
			t.Fatal("expected non-nil client")
		}
		defer client.Close()
	})

	t.Run("AWS IAM with custom env variables", func(t *testing.T) {
		os.Setenv("CUSTOM_ACCESS_KEY", "custom-access")
		os.Setenv("CUSTOM_SECRET_KEY", "custom-secret")
		os.Setenv("CUSTOM_SESSION_TOKEN", "custom-session")
		defer func() {
			os.Unsetenv("CUSTOM_ACCESS_KEY")
			os.Unsetenv("CUSTOM_SECRET_KEY")
			os.Unsetenv("CUSTOM_SESSION_TOKEN")
		}()

		cfg := config.ClusterConfig{
			Name:    "test-aws",
			Brokers: []string{"b-1.msk.amazonaws.com:9098"},
			AWS: &config.AWSConfig{
				IAM:             true,
				Region:          "us-west-2",
				AccessKeyEnv:    "CUSTOM_ACCESS_KEY",
				SecretKeyEnv:    "CUSTOM_SECRET_KEY",
				SessionTokenEnv: "CUSTOM_SESSION_TOKEN",
			},
		}

		client, err := NewClient(cfg)
		if err != nil {
			t.Fatalf("NewClient() with custom AWS env error = %v", err)
		}
		if client == nil {
			t.Fatal("expected non-nil client")
		}
		defer client.Close()
	})

	t.Run("AWS IAM without credentials", func(t *testing.T) {
		// Clear any AWS env vars
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")

		cfg := config.ClusterConfig{
			Name:    "test-aws",
			Brokers: []string{"b-1.msk.amazonaws.com:9098"},
			AWS: &config.AWSConfig{
				IAM:    true,
				Region: "us-east-1",
			},
		}

		client, err := NewClient(cfg)
		if err != nil {
			t.Fatalf("NewClient() without AWS credentials error = %v", err)
		}
		if client == nil {
			t.Fatal("expected non-nil client")
		}
		defer client.Close()
	})
}

func TestClientMethods(t *testing.T) {
	cfg := config.ClusterConfig{
		Name:    "test",
		Brokers: []string{"localhost:9092"},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	t.Run("GetConfig", func(t *testing.T) {
		c := client.GetConfig()
		if c.Name != "test" {
			t.Errorf("expected name 'test', got '%s'", c.Name)
		}
	})

	t.Run("IsHealthy - cannot connect", func(t *testing.T) {
		// This will fail to connect to localhost:9092 in test environment
		healthy := client.IsHealthy()
		// We don't assert the result since it depends on whether Kafka is running
		_ = healthy
	})

	t.Run("ListTopics", func(t *testing.T) {
		topics, err := client.ListTopics(false)
		// May fail if no Kafka is running, which is fine for unit tests
		_ = topics
		_ = err
	})

	t.Run("GetClusterInfo", func(t *testing.T) {
		info, err := client.GetClusterInfo()
		_ = info
		_ = err
	})

	t.Run("GetClusterStats", func(t *testing.T) {
		stats, err := client.GetClusterStats()
		_ = stats
		_ = err
	})

	t.Run("GetBrokerDetails", func(t *testing.T) {
		brokers, err := client.GetBrokerDetails()
		_ = brokers
		_ = err
	})

	t.Run("ListConsumerGroups", func(t *testing.T) {
		groups, err := client.ListConsumerGroups()
		_ = groups
		_ = err
	})

	t.Run("GetTopicDetail", func(t *testing.T) {
		detail, err := client.GetTopicDetail("test-topic")
		_ = detail
		_ = err
	})

	t.Run("CreateTopic", func(t *testing.T) {
		req := domain.CreateTopicRequest{
			Name:              "test-topic",
			NumPartitions:     3,
			ReplicationFactor: 1,
		}
		err := client.CreateTopic(req)
		_ = err
	})

	t.Run("DeleteTopic", func(t *testing.T) {
		err := client.DeleteTopic("test-topic")
		_ = err
	})

	t.Run("UpdateTopicConfig", func(t *testing.T) {
		val := "1000"
		req := domain.UpdateTopicConfigRequest{
			Configs: map[string]*string{
				"retention.ms": &val,
			},
		}
		err := client.UpdateTopicConfig("test-topic", req)
		_ = err
	})

	t.Run("IncreasePartitions", func(t *testing.T) {
		req := domain.IncreasePartitionsRequest{
			TotalPartitions: 5,
		}
		err := client.IncreasePartitions("test-topic", req)
		_ = err
	})
}

func TestClientNilSafety(t *testing.T) {
	var client *Client

	t.Run("IsHealthy on nil client", func(t *testing.T) {
		healthy := client.IsHealthy()
		if healthy {
			t.Error("expected false for nil client")
		}
	})

	t.Run("ListTopics on nil client", func(t *testing.T) {
		topics, err := client.ListTopics(false)
		if topics != nil || err != nil {
			t.Error("expected nil for nil client")
		}
	})

	t.Run("GetClusterInfo on nil client", func(t *testing.T) {
		info, err := client.GetClusterInfo()
		if info != nil || err != nil {
			t.Error("expected nil for nil client")
		}
	})

	t.Run("GetClusterStats on nil client", func(t *testing.T) {
		stats, err := client.GetClusterStats()
		if stats != nil || err != nil {
			t.Error("expected nil for nil client")
		}
	})

	t.Run("Close on nil client", func(t *testing.T) {
		// Should not panic
		client.Close()
	})
}

func TestBuildTLSConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate test certificates
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test CA"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}

	caFile := filepath.Join(tmpDir, "ca.pem")
	caOut, err := os.Create(caFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := pem.Encode(caOut, &pem.Block{Type: "CERTIFICATE", Bytes: caCertDER}); err != nil {
		t.Fatal(err)
	}
	caOut.Close()

	// Generate client certificate
	clientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Test Client"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(24 * time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
		},
	}

	clientCertDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caTemplate, &clientKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}

	certFile := filepath.Join(tmpDir, "client.pem")
	certOut, err := os.Create(certFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: clientCertDER}); err != nil {
		t.Fatal(err)
	}
	certOut.Close()

	keyFile := filepath.Join(tmpDir, "client-key.pem")
	keyOut, err := os.Create(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(clientKey)}); err != nil {
		t.Fatal(err)
	}
	keyOut.Close()

	t.Run("TLS with CA only", func(t *testing.T) {
		tlsCfg := &config.TLSConfig{
			Enabled: true,
			CAFile:  caFile,
		}

		cfg, err := buildTLSConfig(tlsCfg)
		if err != nil {
			t.Fatalf("buildTLSConfig() error = %v", err)
		}
		if cfg == nil {
			t.Fatal("expected non-nil TLS config")
		}
		if cfg.RootCAs == nil {
			t.Error("expected RootCAs to be set")
		}
		if len(cfg.Certificates) != 0 {
			t.Error("expected no client certificates")
		}
	})

	t.Run("mTLS with CA and client certs", func(t *testing.T) {
		tlsCfg := &config.TLSConfig{
			Enabled:  true,
			CAFile:   caFile,
			CertFile: certFile,
			KeyFile:  keyFile,
		}

		cfg, err := buildTLSConfig(tlsCfg)
		if err != nil {
			t.Fatalf("buildTLSConfig() error = %v", err)
		}
		if cfg == nil {
			t.Fatal("expected non-nil TLS config")
		}
		if len(cfg.Certificates) != 1 {
			t.Errorf("expected 1 client certificate, got %d", len(cfg.Certificates))
		}
	})

	t.Run("TLS with insecure skip verify", func(t *testing.T) {
		tlsCfg := &config.TLSConfig{
			Enabled:            true,
			InsecureSkipVerify: true,
		}

		cfg, err := buildTLSConfig(tlsCfg)
		if err != nil {
			t.Fatalf("buildTLSConfig() error = %v", err)
		}
		if !cfg.InsecureSkipVerify {
			t.Error("expected InsecureSkipVerify to be true")
		}
	})

	t.Run("TLS with invalid CA file", func(t *testing.T) {
		tlsCfg := &config.TLSConfig{
			Enabled: true,
			CAFile:  "/nonexistent/ca.pem",
		}

		_, err := buildTLSConfig(tlsCfg)
		if err == nil {
			t.Error("expected error for invalid CA file")
		}
	})

	t.Run("TLS with invalid cert/key pair", func(t *testing.T) {
		tlsCfg := &config.TLSConfig{
			Enabled:  true,
			CertFile: "/nonexistent/cert.pem",
			KeyFile:  "/nonexistent/key.pem",
		}

		_, err := buildTLSConfig(tlsCfg)
		if err == nil {
			t.Error("expected error for invalid cert/key pair")
		}
	})
}

func TestBuildSASLMechanism(t *testing.T) {
	t.Run("PLAIN mechanism", func(t *testing.T) {
		saslCfg := &config.SASLConfig{
			Mechanism: "PLAIN",
			Username:  "user",
			Password:  "pass",
		}

		mech, err := buildSASLMechanism(saslCfg)
		if err != nil {
			t.Fatalf("buildSASLMechanism() error = %v", err)
		}
		if mech == nil {
			t.Error("expected non-nil mechanism")
		}
	})

	t.Run("SCRAM-SHA-256 mechanism", func(t *testing.T) {
		saslCfg := &config.SASLConfig{
			Mechanism: "SCRAM-SHA-256",
			Username:  "user",
			Password:  "pass",
		}

		mech, err := buildSASLMechanism(saslCfg)
		if err != nil {
			t.Fatalf("buildSASLMechanism() error = %v", err)
		}
		if mech == nil {
			t.Error("expected non-nil mechanism")
		}
	})

	t.Run("SCRAM-SHA-512 mechanism", func(t *testing.T) {
		saslCfg := &config.SASLConfig{
			Mechanism: "SCRAM-SHA-512",
			Username:  "user",
			Password:  "pass",
		}

		mech, err := buildSASLMechanism(saslCfg)
		if err != nil {
			t.Fatalf("buildSASLMechanism() error = %v", err)
		}
		if mech == nil {
			t.Error("expected non-nil mechanism")
		}
	})

	t.Run("mechanism with env variables", func(t *testing.T) {
		os.Setenv("SASL_USER", "envuser")
		os.Setenv("SASL_PASS", "envpass")
		defer func() {
			os.Unsetenv("SASL_USER")
			os.Unsetenv("SASL_PASS")
		}()

		saslCfg := &config.SASLConfig{
			Mechanism:   "PLAIN",
			UsernameEnv: "SASL_USER",
			PasswordEnv: "SASL_PASS",
		}

		mech, err := buildSASLMechanism(saslCfg)
		if err != nil {
			t.Fatalf("buildSASLMechanism() error = %v", err)
		}
		if mech == nil {
			t.Error("expected non-nil mechanism")
		}
	})

	t.Run("unknown mechanism", func(t *testing.T) {
		saslCfg := &config.SASLConfig{
			Mechanism: "UNKNOWN",
			Username:  "user",
			Password:  "pass",
		}

		mech, err := buildSASLMechanism(saslCfg)
		if err != nil {
			t.Fatalf("buildSASLMechanism() error = %v", err)
		}
		if mech != nil {
			t.Error("expected nil mechanism for unknown type")
		}
	})

	t.Run("case insensitive mechanisms", func(t *testing.T) {
		mechanisms := []string{"plain", "PLAIN", "scram-sha-256", "SCRAM-SHA256"}
		for _, m := range mechanisms {
			saslCfg := &config.SASLConfig{
				Mechanism: m,
				Username:  "user",
				Password:  "pass",
			}

			mech, err := buildSASLMechanism(saslCfg)
			if err != nil {
				t.Errorf("buildSASLMechanism(%s) error = %v", m, err)
			}
			if mech == nil {
				t.Errorf("expected non-nil mechanism for %s", m)
			}
		}
	})
}

func TestBuildAWSMechanism(t *testing.T) {
	t.Run("AWS with default env variables", func(t *testing.T) {
		os.Setenv("AWS_ACCESS_KEY_ID", "test-access")
		os.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")
		defer func() {
			os.Unsetenv("AWS_ACCESS_KEY_ID")
			os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		}()

		awsCfg := &config.AWSConfig{
			IAM:    true,
			Region: "us-east-1",
		}

		mech, err := buildAWSMechanism(awsCfg)
		if err != nil {
			t.Fatalf("buildAWSMechanism() error = %v", err)
		}
		if mech == nil {
			t.Error("expected non-nil mechanism")
		}
	})

	t.Run("AWS with custom env variables", func(t *testing.T) {
		os.Setenv("CUSTOM_ACCESS", "custom-access")
		os.Setenv("CUSTOM_SECRET", "custom-secret")
		os.Setenv("CUSTOM_SESSION", "custom-session")
		defer func() {
			os.Unsetenv("CUSTOM_ACCESS")
			os.Unsetenv("CUSTOM_SECRET")
			os.Unsetenv("CUSTOM_SESSION")
		}()

		awsCfg := &config.AWSConfig{
			IAM:             true,
			Region:          "us-west-2",
			AccessKeyEnv:    "CUSTOM_ACCESS",
			SecretKeyEnv:    "CUSTOM_SECRET",
			SessionTokenEnv: "CUSTOM_SESSION",
		}

		mech, err := buildAWSMechanism(awsCfg)
		if err != nil {
			t.Fatalf("buildAWSMechanism() error = %v", err)
		}
		if mech == nil {
			t.Error("expected non-nil mechanism")
		}
	})

	t.Run("AWS without credentials", func(t *testing.T) {
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")

		awsCfg := &config.AWSConfig{
			IAM:    true,
			Region: "us-east-1",
		}

		mech, err := buildAWSMechanism(awsCfg)
		if err != nil {
			t.Fatalf("buildAWSMechanism() error = %v", err)
		}
		if mech != nil {
			t.Error("expected nil mechanism without credentials")
		}
	})

	t.Run("AWS with only access key", func(t *testing.T) {
		os.Setenv("AWS_ACCESS_KEY_ID", "test-access")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		defer os.Unsetenv("AWS_ACCESS_KEY_ID")

		awsCfg := &config.AWSConfig{
			IAM:    true,
			Region: "us-east-1",
		}

		mech, err := buildAWSMechanism(awsCfg)
		if err != nil {
			t.Fatalf("buildAWSMechanism() error = %v", err)
		}
		if mech != nil {
			t.Error("expected nil mechanism with incomplete credentials")
		}
	})

	t.Run("AWS with nil config", func(t *testing.T) {
		os.Setenv("AWS_ACCESS_KEY_ID", "test-access")
		os.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")
		defer func() {
			os.Unsetenv("AWS_ACCESS_KEY_ID")
			os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		}()

		mech, err := buildAWSMechanism(nil)
		if err != nil {
			t.Fatalf("buildAWSMechanism() error = %v", err)
		}
		if mech == nil {
			t.Error("expected non-nil mechanism with default env vars")
		}
	})
}
