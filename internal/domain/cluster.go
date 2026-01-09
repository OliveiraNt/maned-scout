// Package domain defines the core business entities and interfaces for the Maned Scout application.
// It includes data structures for Kafka clusters, topics, consumer groups, brokers, and their associated
// statistics, as well as abstractions for Kafka client operations and client factory creation.
package domain

import "github.com/OliveiraNt/maned-scout/internal/config"

// Cluster represents a Kafka cluster with its metadata
type Cluster struct {
	ID       string                  `json:"id"`
	Name     string                  `json:"name"`
	Brokers  []string                `json:"brokers"`
	IsOnline bool                    `json:"is_online"`
	AuthType string                  `json:"auth_type"`
	CertInfo *config.CertificateInfo `json:"cert_info,omitempty"`
}

// ClusterStats holds detailed statistics about a cluster
type ClusterStats struct {
	TotalTopics               int `json:"total_topics"`
	TotalPartitions           int `json:"total_partitions"`
	TotalConsumerGroups       int `json:"total_consumer_groups"`
	UnderReplicatedPartitions int `json:"under_replicated_partitions"`
	OfflinePartitions         int `json:"offline_partitions"`
}

// BrokerDetail holds detailed information about a broker
type BrokerDetail struct {
	ID               int32  `json:"id"`
	Host             string `json:"host"`
	Port             int32  `json:"port"`
	Rack             string `json:"rack"`
	IsController     bool   `json:"is_controller"`
	LeaderPartitions int    `json:"leader_partitions"`
}

// ConsumerGroupSummary holds basic info about a consumer group
type ConsumerGroupSummary struct {
	GroupID string `json:"group_id"`
	State   string `json:"state"`
	Members int    `json:"members"`
}
