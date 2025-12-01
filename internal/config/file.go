package config

import (
	"os"

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
	Mechanism      string `yaml:"mechanism,omitempty" json:"mechanism,omitempty"` // e.g. PLAIN, SCRAM-SHA-256, SCRAM-SHA-512
	Username       string `yaml:"username,omitempty" json:"username,omitempty"`
	Password       string `yaml:"password,omitempty" json:"password,omitempty"`
	UsernameEnv    string `yaml:"username_env,omitempty" json:"username_env,omitempty"`
	PasswordEnv    string `yaml:"password_env,omitempty" json:"password_env,omitempty"`
	ScramAlgorithm string `yaml:"scram_algorithm,omitempty" json:"scram_algorithm,omitempty"` // optional explicit algorithm
}

// AWSConfig holds AWS IAM SASL config. Prefer the standard AWS credential provider (env, shared creds, role).
type AWSConfig struct {
	IAM             bool   `yaml:"iam,omitempty" json:"iam,omitempty"`
	Region          string `yaml:"region,omitempty" json:"region,omitempty"`
	AccessKeyEnv    string `yaml:"access_key_env,omitempty" json:"access_key_env,omitempty"`
	SecretKeyEnv    string `yaml:"secret_key_env,omitempty" json:"secret_key_env,omitempty"`
	SessionTokenEnv string `yaml:"session_token_env,omitempty" json:"session_token_env,omitempty"`
}

type FileConfig struct {
	Clusters []ClusterConfig `yaml:"clusters" json:"clusters"`
}

func ReadConfig(path string) (FileConfig, error) {
	var cfg FileConfig
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	err = yaml.Unmarshal(b, &cfg)
	return cfg, err
}

func WriteConfig(path string, cfg FileConfig) error {
	b, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}
