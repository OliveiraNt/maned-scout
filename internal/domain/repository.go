package domain

import (
	"context"

	"github.com/OliveiraNt/maned-scout/internal/config"
	"github.com/twmb/franz-go/pkg/kadm"
)

// ClusterRepository defines operations for managing cluster configurations.
type ClusterRepository interface {
	Save(cfg config.ClusterConfig) error
	Delete(name string) error
	FindByName(name string) (config.ClusterConfig, bool)
	FindAll() []config.ClusterConfig
	Watch() error
	GetClient(name string) (KafkaClient, bool)
}

// ClientFactory creates Kafka clients from configuration.
type ClientFactory interface {
	CreateClient(cfg config.ClusterConfig) (KafkaClient, error)
}

// KafkaClient defines operations for interacting with a Kafka cluster.
type KafkaClient interface {
	IsHealthy() bool
	ListTopics(showInternal bool) (map[string]int, error)
	GetClusterInfo() (*Cluster, error)
	GetClusterStats() (*ClusterStats, error)
	GetBrokerDetails() ([]BrokerDetail, error)
	ListConsumerGroups() ([]ConsumerGroupSummary, error)
	ListConsumerGroupsWithLagFromTopic(ctx context.Context, topicName string) (kadm.DescribedGroupLags, error)
	GetTopicDetail(topicName string) (*TopicDetail, error)
	CreateTopic(req CreateTopicRequest) error
	DeleteTopic(topicName string) error
	UpdateTopicConfig(topicName string, req UpdateTopicConfigRequest) error
	IncreasePartitions(topicName string, req IncreasePartitionsRequest) error
	StreamMessages(ctx context.Context, topic string, out chan<- Message)
	WriteMessage(ctx context.Context, topic string, msg Message)
	Close()
}
