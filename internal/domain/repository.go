package domain

import (
	"context"

	"github.com/OliveiraNt/kdash/internal/config"
)

// ClusterRepository defines operations for managing cluster configurations.
type ClusterRepository interface {
	Save(cfg config.ClusterConfig) error
	Delete(name string) error
	FindByName(name string) (config.ClusterConfig, bool)
	FindAll() []config.ClusterConfig
	Watch() error
}

// KafkaClient defines operations for interacting with a Kafka cluster.
type KafkaClient interface {
	IsHealthy() bool
	ListTopics(showInternal bool) (map[string]int, error)
	GetClusterInfo() (*Cluster, error)
	GetClusterStats() (*ClusterStats, error)
	GetBrokerDetails() ([]BrokerDetail, error)
	ListConsumerGroups() ([]ConsumerGroupSummary, error)
	GetTopicDetail(topicName string) (*TopicDetail, error)
	CreateTopic(req CreateTopicRequest) error
	DeleteTopic(topicName string) error
	UpdateTopicConfig(topicName string, req UpdateTopicConfigRequest) error
	IncreasePartitions(topicName string, req IncreasePartitionsRequest) error
	StreamMessages(ctx context.Context, topic string, out chan<- Message)
	Close()
}

// KafkaAdmin defines low-level Kafka admin operations.
type KafkaAdmin interface {
	ListTopics(ctx context.Context, showInternal bool) (map[string]int, error)
	GetClusterInfo(ctx context.Context) (*Cluster, error)
	GetClusterStats(ctx context.Context) (*ClusterStats, error)
	GetBrokerDetails(ctx context.Context) ([]BrokerDetail, error)
	ListConsumerGroups(ctx context.Context) ([]ConsumerGroupSummary, error)
}
