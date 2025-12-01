package kafka

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"
	"time"

	"github.com/OliveiraNt/kdash/internal/config"
	"github.com/OliveiraNt/kdash/internal/core"
	adminpkg "github.com/OliveiraNt/kdash/internal/infra/kafka/admin"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl"
	"github.com/twmb/franz-go/pkg/sasl/aws"
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"github.com/twmb/franz-go/pkg/sasl/scram"
)

type ClientWrapper struct {
	Client  *kgo.Client
	Admin   *kadm.Client
	Brokers []string
	Config  config.ClusterConfig
}

// NewKafkaClient creates a kafka client based on the provided config, supporting TLS, SASL and AWS IAM.
func NewKafkaClient(cfg config.ClusterConfig) (*ClientWrapper, error) {
	var opts []kgo.Opt
	if cfg.ClientID != "" {
		opts = append(opts, kgo.ClientID(cfg.ClientID))
	}
	// seed brokers
	if len(cfg.Brokers) > 0 {
		opts = append(opts, kgo.SeedBrokers(cfg.Brokers...))
	}

	// TLS
	if cfg.TLS != nil && cfg.TLS.Enabled {
		tlsCfg, err := buildTLSConfig(cfg.TLS)
		if err != nil {
			return nil, err
		}
		opts = append(opts, kgo.DialTLSConfig(tlsCfg))
	}

	// SASL
	if cfg.SASL != nil && cfg.SASL.Mechanism != "" {
		mech, err := buildSASLMechanism(cfg.SASL)
		if err != nil {
			return nil, err
		}
		if mech != nil {
			opts = append(opts, kgo.SASL(mech))
		}
	}

	// AWS IAM
	if cfg.AWS != nil && cfg.AWS.IAM {
		awsMech, err := buildAWSMechanism(cfg.AWS)
		if err != nil {
			return nil, err
		}
		if awsMech != nil {
			opts = append(opts, kgo.SASL(awsMech))
		}
	}

	cl, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, err
	}
	admin := kadm.NewClient(cl)
	return &ClientWrapper{
		Client:  cl,
		Admin:   admin,
		Brokers: cfg.Brokers,
		Config:  cfg,
	}, nil
}

// buildTLSConfig reads cert files and builds a tls.Config.
func buildTLSConfig(t *config.TLSConfig) (*tls.Config, error) {
	rootCAs := x509.NewCertPool()
	if t.CAFile != "" {
		b, err := os.ReadFile(t.CAFile)
		if err != nil {
			return nil, err
		}
		if ok := rootCAs.AppendCertsFromPEM(b); !ok {
			// proceed even if not ok
		}
	}
	var cert tls.Certificate
	if t.CertFile != "" && t.KeyFile != "" {
		c, err := tls.LoadX509KeyPair(t.CertFile, t.KeyFile)
		if err != nil {
			return nil, err
		}
		cert = c
	}
	cfg := &tls.Config{
		RootCAs:            rootCAs,
		InsecureSkipVerify: t.InsecureSkipVerify,
	}
	if len(cert.Certificate) > 0 {
		cfg.Certificates = []tls.Certificate{cert}
	}
	return cfg, nil
}

// buildSASLMechanism creates a franz-go sasl.Mechanism based on SASLConfig.
func buildSASLMechanism(s *config.SASLConfig) (sasl.Mechanism, error) {
	mech := s.Mechanism
	// resolve credentials: env takes precedence
	username := s.Username
	password := s.Password
	if s.UsernameEnv != "" {
		if v := os.Getenv(s.UsernameEnv); v != "" {
			username = v
		}
	}
	if s.PasswordEnv != "" {
		if v := os.Getenv(s.PasswordEnv); v != "" {
			password = v
		}
	}

	switch mech {
	case "PLAIN", "plain":
		// use plain package
		return plain.Auth{User: username, Pass: password}.AsMechanism(), nil
	case "SCRAM-SHA-256", "SCRAM-SHA256", "scram-sha-256":
		return scram.Auth{User: username, Pass: password}.AsSha256Mechanism(), nil
	case "SCRAM-SHA-512", "SCRAM-SHA512", "scram-sha-512":
		return scram.Auth{User: username, Pass: password}.AsSha512Mechanism(), nil
	default:
		return nil, nil // unknown mechanism, nil means no SASL
	}
}

// buildAWSMechanism constructs an AWS IAM SASL mechanism using environment credentials or defaults.
func buildAWSMechanism(a *config.AWSConfig) (sasl.Mechanism, error) {
	// resolve credentials from configured env var names or default AWS env vars
	access := ""
	secret := ""
	session := ""
	if a != nil {
		if a.AccessKeyEnv != "" {
			access = os.Getenv(a.AccessKeyEnv)
		}
		if a.SecretKeyEnv != "" {
			secret = os.Getenv(a.SecretKeyEnv)
		}
		if a.SessionTokenEnv != "" {
			session = os.Getenv(a.SessionTokenEnv)
		}
	}
	// fallback to standard AWS env vars
	if access == "" {
		access = os.Getenv("AWS_ACCESS_KEY_ID")
	}
	if secret == "" {
		secret = os.Getenv("AWS_SECRET_ACCESS_KEY")
	}
	if session == "" {
		session = os.Getenv("AWS_SESSION_TOKEN")
	}
	if access == "" || secret == "" {
		// no creds available; return nil to indicate we couldn't build mech
		return nil, nil
	}
	return aws.Auth{AccessKey: access, SecretKey: secret, SessionToken: session}.AsManagedStreamingIAMMechanism(), nil
}

func (w *ClientWrapper) Close() {
	if w == nil {
		return
	}
	// close client (kgo has Close)
	w.Client.Close()
}

// IsHealthy checks if the cluster is reachable by attempting to fetch broker metadata.
func (w *ClientWrapper) IsHealthy() bool {
	if w == nil || w.Admin == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := adminpkg.NewKadmAdmin(w.Admin).BrokerMetadata(ctx)
	return err == nil
}

// ListTopics delegates to admin package to list topics using the admin client.
func (w *ClientWrapper) ListTopics() (map[string]int, error) {
	if w == nil || w.Admin == nil {
		return nil, nil
	}
	return adminpkg.ListTopics(context.Background(), adminpkg.NewKadmAdmin(w.Admin))
}

// GetClusterInfo delegates to admin package to fetch cluster info.
func (w *ClientWrapper) GetClusterInfo() (*core.Cluster, error) {
	if w == nil || w.Admin == nil {
		return nil, nil
	}
	return adminpkg.GetClusterInfo(context.Background(), adminpkg.NewKadmAdmin(w.Admin))
}

// GetClusterStats returns detailed statistics about the cluster
func (w *ClientWrapper) GetClusterStats() (*core.ClusterStats, error) {
	if w == nil || w.Admin == nil {
		return nil, nil
	}
	return adminpkg.GetClusterStats(context.Background(), adminpkg.NewKadmAdmin(w.Admin))
}

// GetBrokerDetails returns detailed information about all brokers
func (w *ClientWrapper) GetBrokerDetails() ([]core.BrokerDetail, error) {
	if w == nil || w.Admin == nil {
		return nil, nil
	}
	return adminpkg.GetBrokerDetails(context.Background(), adminpkg.NewKadmAdmin(w.Admin))
}

// ListConsumerGroups returns a list of consumer groups
func (w *ClientWrapper) ListConsumerGroups() ([]core.ConsumerGroupSummary, error) {
	if w == nil || w.Admin == nil {
		return nil, nil
	}
	return adminpkg.ListConsumerGroups(context.Background(), adminpkg.NewKadmAdmin(w.Admin))
}
