// Package config provides configuration management for Kafka cluster connectivity.
// It handles loading, parsing, and persisting cluster configurations from YAML files,
// including support for TLS, SASL, and AWS IAM authentication mechanisms.
// The package also provides utilities for extracting certificate information and
// determining authentication types for configured clusters.
package config

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// ClusterConfig holds cluster connectivity and security configuration.
type ClusterConfig struct {
	Name     string            `yaml:"name" json:"name"`
	Brokers  []string          `yaml:"brokers" json:"brokers"`
	ClientID string            `yaml:"client_id,omitempty" json:"client_id,omitempty"`
	TLS      *TLSConfig        `yaml:"tls,omitempty" json:"tls,omitempty"`
	SASL     *SASLConfig       `yaml:"sasl,omitempty" json:"sasl,omitempty"`
	AWS      *AWSConfig        `yaml:"aws,omitempty" json:"aws,omitempty"`
	Options  map[string]string `yaml:"options,omitempty" json:"options,omitempty"`
}

// TLSConfig holds TLS related fields.
type TLSConfig struct {
	Enabled            bool   `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	CAFile             string `yaml:"ca_file,omitempty" json:"ca_file,omitempty"`
	CertFile           string `yaml:"cert_file,omitempty" json:"cert_file,omitempty"`
	KeyFile            string `yaml:"key_file,omitempty" json:"key_file,omitempty"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify,omitempty" json:"insecure_skip_verify,omitempty"`
}

// SASLConfig holds SASL configuration. Credentials may be provided inline or via env var names.
type SASLConfig struct {
	Mechanism      string `yaml:"mechanism,omitempty" json:"mechanism,omitempty"`
	Username       string `yaml:"username,omitempty" json:"username,omitempty"`
	Password       string `yaml:"password,omitempty" json:"password,omitempty"`
	UsernameEnv    string `yaml:"username_env,omitempty" json:"username_env,omitempty"`
	PasswordEnv    string `yaml:"password_env,omitempty" json:"password_env,omitempty"`
	ScramAlgorithm string `yaml:"scram_algorithm,omitempty" json:"scram_algorithm,omitempty"`
}

// AWSConfig holds AWS IAM SASL config. Prefer the standard AWS credential provider (env, shared creds, role).
type AWSConfig struct {
	IAM             bool   `yaml:"iam,omitempty" json:"iam,omitempty"`
	Region          string `yaml:"region,omitempty" json:"region,omitempty"`
	AccessKeyEnv    string `yaml:"access_key_env,omitempty" json:"access_key_env,omitempty"`
	SecretKeyEnv    string `yaml:"secret_key_env,omitempty" json:"secret_key_env,omitempty"`
	SessionTokenEnv string `yaml:"session_token_env,omitempty" json:"session_token_env,omitempty"`
}

// FileConfig represents the root configuration file structure for Maned Scout.
type FileConfig struct {
	Clusters []ClusterConfig `yaml:"clusters" json:"clusters"`
}

// ReadConfig loads a FileConfig from the provided path.
func ReadConfig(path string) (FileConfig, error) {
	var cfg FileConfig
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	err = yaml.Unmarshal(b, &cfg)
	return cfg, err
}

// WriteConfig persists the FileConfig to the provided path.
func WriteConfig(path string, cfg FileConfig) error {
	b, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

// GetAuthType returns a human-readable authentication type based on the cluster config.
func (c *ClusterConfig) GetAuthType() string {
	if c.AWS != nil && c.AWS.IAM {
		return "AWS IAM"
	}
	if c.SASL != nil && c.SASL.Mechanism != "" {
		mechanism := c.SASL.Mechanism
		if c.TLS != nil && c.TLS.Enabled {
			return "SASL/" + mechanism + " + TLS"
		}
		return "SASL/" + mechanism
	}
	if c.TLS != nil && c.TLS.Enabled {
		if c.TLS.CertFile != "" && c.TLS.KeyFile != "" {
			return "mTLS"
		}
		return "TLS"
	}
	return "PLAINTEXT"
}

// CertificateInfo holds certificate validity information.
type CertificateInfo struct {
	NotBefore    time.Time `json:"not_before"`
	NotAfter     time.Time `json:"not_after"`
	DaysToExpiry int       `json:"days_to_expiry"`
	Status       string    `json:"status"`
}

// GetCertificateInfo reads and parses the certificate file to extract validity information.
func (c *ClusterConfig) GetCertificateInfo() (*CertificateInfo, error) {
	if c.TLS == nil || !c.TLS.Enabled || c.TLS.CertFile == "" {
		return nil, nil
	}
	certPEM, err := os.ReadFile(c.TLS.CertFile)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, nil
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	daysToExpiry := int(time.Until(cert.NotAfter).Hours() / 24)
	status := "valid"
	if now.After(cert.NotAfter) {
		status = "expired"
	} else if daysToExpiry <= 7 {
		status = "critical"
	} else if daysToExpiry <= 30 {
		status = "warning"
	}

	return &CertificateInfo{
		NotBefore:    cert.NotBefore,
		NotAfter:     cert.NotAfter,
		DaysToExpiry: daysToExpiry,
		Status:       status,
	}, nil
}

// HasCertificate returns true if the cluster uses certificate-based authentication.
func (c *ClusterConfig) HasCertificate() bool {
	return c.TLS != nil && c.TLS.Enabled && c.TLS.CertFile != ""
}
