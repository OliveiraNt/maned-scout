package domain

import (
	"context"

	"github.com/OliveiraNt/kdash/internal/config"
)

// ClusterRepository defines operations for managing cluster configurations
type ClusterRepository interface {
	// Save persists a cluster configuration
	Save(cfg config.ClusterConfig) error

	// Delete removes a cluster configuration by name
	Delete(name string) error

	// FindByName retrieves a cluster configuration by name
	FindByName(name string) (config.ClusterConfig, bool)

	// FindAll retrieves all cluster configurations
	FindAll() []config.ClusterConfig

	// Watch monitors configuration changes
	Watch() error
}

// KafkaClient defines operations for interacting with a Kafka cluster
type KafkaClient interface {
	// IsHealthy checks if the cluster is reachable
	IsHealthy() bool

	// ListTopics returns topics with partition counts
	ListTopics(showInternal bool) (map[string]int, error)

	// GetClusterInfo returns cluster information
	GetClusterInfo() (*Cluster, error)

	// GetClusterStats returns cluster statistics
	GetClusterStats() (*ClusterStats, error)

	// GetBrokerDetails returns broker information
	GetBrokerDetails() ([]BrokerDetail, error)

	// ListConsumerGroups returns consumer group information
	ListConsumerGroups() ([]ConsumerGroupSummary, error)

	// GetTopicDetail returns detailed information about a topic
	GetTopicDetail(topicName string) (*TopicDetail, error)

	// CreateTopic creates a new topic
	CreateTopic(req CreateTopicRequest) error

	// DeleteTopic deletes a topic
	DeleteTopic(topicName string) error

	// UpdateTopicConfig updates topic configurations
	UpdateTopicConfig(topicName string, req UpdateTopicConfigRequest) error

	// IncreasePartitions increases the number of partitions for a topic
	IncreasePartitions(topicName string, req IncreasePartitionsRequest) error

	// Close releases resources
	Close()
}

// KafkaAdmin defines low-level Kafka admin operations
type KafkaAdmin interface {
	// ListTopics lists topics
	ListTopics(ctx context.Context, showInternal bool) (map[string]int, error)

	// GetClusterInfo retrieves cluster metadata
	GetClusterInfo(ctx context.Context) (*Cluster, error)

	// GetClusterStats retrieves cluster statistics
	GetClusterStats(ctx context.Context) (*ClusterStats, error)

	// GetBrokerDetails retrieves broker details
	GetBrokerDetails(ctx context.Context) ([]BrokerDetail, error)

	// ListConsumerGroups lists consumer groups
	ListConsumerGroups(ctx context.Context) ([]ConsumerGroupSummary, error)
}
