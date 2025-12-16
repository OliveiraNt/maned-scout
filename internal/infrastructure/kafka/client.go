package kafka

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"
	"time"

	"github.com/OliveiraNt/kdash/internal/config"
	"github.com/OliveiraNt/kdash/internal/domain"
	"github.com/OliveiraNt/kdash/internal/registry"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl"
	"github.com/twmb/franz-go/pkg/sasl/aws"
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"github.com/twmb/franz-go/pkg/sasl/scram"
)

// Client implements domain.KafkaClient using franz-go.
type Client struct {
	client *kgo.Client
	admin  *Admin
	config config.ClusterConfig
}

// NewClient creates a new Kafka client from configuration.
func NewClient(cfg config.ClusterConfig) (*Client, error) {
	var opts []kgo.Opt

	if cfg.ClientID != "" {
		opts = append(opts, kgo.ClientID(cfg.ClientID))
	}

	if len(cfg.Brokers) > 0 {
		opts = append(opts, kgo.SeedBrokers(cfg.Brokers...))
	}
	if cfg.TLS != nil && cfg.TLS.Enabled {
		tlsCfg, err := buildTLSConfig(cfg.TLS)
		if err != nil {
			return nil, err
		}
		opts = append(opts, kgo.DialTLSConfig(tlsCfg))
	}
	if cfg.SASL != nil && cfg.SASL.Mechanism != "" {
		mech, err := buildSASLMechanism(cfg.SASL)
		if err != nil {
			return nil, err
		}
		if mech != nil {
			opts = append(opts, kgo.SASL(mech))
		}
	}
	if cfg.AWS != nil && cfg.AWS.IAM {
		awsMech, err := buildAWSMechanism(cfg.AWS)
		if err != nil {
			return nil, err
		}
		if awsMech != nil {
			opts = append(opts, kgo.SASL(awsMech))
		}
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, err
	}

	kadmClient := kadm.NewClient(client)
	admin := NewAdmin(kadmClient)

	return &Client{
		client: client,
		admin:  admin,
		config: cfg,
	}, nil
}

// IsHealthy checks if the cluster is reachable.
func (c *Client) IsHealthy() bool {
	if c == nil || c.admin == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.admin.BrokerMetadata(ctx)
	return err == nil
}

// ListTopics returns topics with partition counts.
func (c *Client) ListTopics(showInternal bool) (map[string]int, error) {
	if c == nil || c.admin == nil {
		return nil, nil
	}
	return c.admin.ListTopics(context.Background(), showInternal)
}

// GetClusterInfo returns cluster information.
func (c *Client) GetClusterInfo() (*domain.Cluster, error) {
	if c == nil || c.admin == nil {
		return nil, nil
	}
	return c.admin.GetClusterInfo(context.Background())
}

// GetClusterStats returns cluster statistics
func (c *Client) GetClusterStats() (*domain.ClusterStats, error) {
	if c == nil || c.admin == nil {
		return nil, nil
	}
	return c.admin.GetClusterStats(context.Background())
}

// GetBrokerDetails returns broker information
func (c *Client) GetBrokerDetails() ([]domain.BrokerDetail, error) {
	if c == nil || c.admin == nil {
		return nil, nil
	}
	return c.admin.GetBrokerDetails(context.Background())
}

// ListConsumerGroups returns consumer group information
func (c *Client) ListConsumerGroups() ([]domain.ConsumerGroupSummary, error) {
	if c == nil || c.admin == nil {
		return nil, nil
	}
	return c.admin.ListConsumerGroups(context.Background())
}

// GetTopicDetail returns detailed information about a topic
func (c *Client) GetTopicDetail(topicName string) (*domain.TopicDetail, error) {
	if c == nil || c.admin == nil {
		return nil, nil
	}
	return c.admin.GetTopicDetail(context.Background(), topicName)
}

// CreateTopic creates a new topic
func (c *Client) CreateTopic(req domain.CreateTopicRequest) error {
	if c == nil || c.admin == nil {
		return nil
	}
	return c.admin.CreateTopic(context.Background(), req)
}

// DeleteTopic deletes a topic
func (c *Client) DeleteTopic(topicName string) error {
	if c == nil || c.admin == nil {
		return nil
	}
	return c.admin.DeleteTopic(context.Background(), topicName)
}

// UpdateTopicConfig updates topic configurations
func (c *Client) UpdateTopicConfig(topicName string, req domain.UpdateTopicConfigRequest) error {
	if c == nil || c.admin == nil {
		return nil
	}
	return c.admin.UpdateTopicConfig(context.Background(), topicName, req)
}

// IncreasePartitions increases the number of partitions for a topic
func (c *Client) IncreasePartitions(topicName string, req domain.IncreasePartitionsRequest) error {
	if c == nil || c.admin == nil {
		return nil
	}
	return c.admin.IncreasePartitions(context.Background(), topicName, req)
}

// Close releases resources
func (c *Client) Close() {
	if c != nil && c.client != nil {
		c.client.Close()
	}
}

// GetConfig returns the cluster configuration
func (c *Client) GetConfig() config.ClusterConfig {
	return c.config
}

func (c *Client) StreamMessages(ctx context.Context, topic string, out chan<- domain.Message) {
	if c == nil || c.client == nil {
		return
	}
	c.client.AddConsumeTopics(topic)

	for {
		if ctx.Err() != nil {
			return
		}
		fetches := c.client.PollFetches(ctx)
		if fetches.IsClientClosed() || ctx.Err() != nil {
			return
		}
		fetches.EachError(func(t string, p int32, err error) {
			registry.Logger.Errorf("Error fetching messages from topic %s partition %d: %v", t, p, err)
		})
		fetches.EachRecord(func(r *kgo.Record) {
			select {
			case out <- domain.Message{
				Key:       r.Key,
				Value:     r.Value,
				Timestamp: r.Timestamp,
				Partition: r.Partition,
				Offset:    r.Offset,
			}:
			case <-ctx.Done():
				return
			}
		})
	}
}

// buildTLSConfig reads cert files and builds a tls.Config
func buildTLSConfig(t *config.TLSConfig) (*tls.Config, error) {
	rootCAs := x509.NewCertPool()
	if t.CAFile != "" {
		b, err := os.ReadFile(t.CAFile)
		if err != nil {
			return nil, err
		}
		rootCAs.AppendCertsFromPEM(b)
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

// buildSASLMechanism creates a franz-go sasl.Mechanism based on SASLConfig
func buildSASLMechanism(s *config.SASLConfig) (sasl.Mechanism, error) {
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

	switch s.Mechanism {
	case "PLAIN", "plain":
		return plain.Auth{User: username, Pass: password}.AsMechanism(), nil
	case "SCRAM-SHA-256", "SCRAM-SHA256", "scram-sha-256":
		return scram.Auth{User: username, Pass: password}.AsSha256Mechanism(), nil
	case "SCRAM-SHA-512", "SCRAM-SHA512", "scram-sha-512":
		return scram.Auth{User: username, Pass: password}.AsSha512Mechanism(), nil
	default:
		return nil, nil
	}
}

// buildAWSMechanism constructs an AWS IAM SASL mechanism
func buildAWSMechanism(a *config.AWSConfig) (sasl.Mechanism, error) {
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
		return nil, nil
	}

	return aws.Auth{
		AccessKey:    access,
		SecretKey:    secret,
		SessionToken: session,
	}.AsManagedStreamingIAMMechanism(), nil
}
