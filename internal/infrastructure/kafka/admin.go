package kafka

import (
	"context"
	"strconv"
	"time"

	"github.com/OliveiraNt/kdash/internal/domain"
	"github.com/twmb/franz-go/pkg/kadm"
)

// Admin implements domain.KafkaAdmin using franz-go kadm
type Admin struct {
	client *kadm.Client
}

// NewAdmin creates a new Admin
func NewAdmin(client *kadm.Client) *Admin {
	return &Admin{client: client}
}

// BrokerMetadata returns broker metadata (used for health checks)
func (a *Admin) BrokerMetadata(ctx context.Context) (kadm.Metadata, error) {
	return a.client.BrokerMetadata(ctx)
}

// ListTopics returns topics as a simplified map name->partitions
func (a *Admin) ListTopics(ctx context.Context, showInternal bool) (map[string]int, error) {
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var m kadm.TopicDetails
	var err error

	if showInternal {
		m, err = a.client.ListTopicsWithInternal(cctx)
	} else {
		m, err = a.client.ListTopics(cctx)
	}

	if err != nil {
		return nil, err
	}

	out := make(map[string]int)
	for name, info := range m {
		out[name] = len(info.Partitions)
	}
	return out, nil
}

// GetClusterInfo returns cluster information
func (a *Admin) GetClusterInfo(ctx context.Context) (*domain.Cluster, error) {
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	meta, err := a.client.BrokerMetadata(cctx)
	if err != nil {
		return nil, err
	}

	brokers := make([]string, len(meta.Brokers))
	for i, b := range meta.Brokers {
		brokers[i] = b.Host + ":" + strconv.Itoa(int(b.Port))
	}

	return &domain.Cluster{
		ID:      meta.Cluster,
		Brokers: brokers,
	}, nil
}

// GetClusterStats returns detailed statistics about the cluster
func (a *Admin) GetClusterStats(ctx context.Context) (*domain.ClusterStats, error) {
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	stats := &domain.ClusterStats{}

	topics, err := a.client.ListTopics(cctx)
	if err != nil {
		return nil, err
	}

	totalPartitions := 0
	underReplicated := 0
	offline := 0

	for _, topic := range topics {
		if topic.IsInternal {
			continue
		}
		stats.TotalTopics++

		for _, partition := range topic.Partitions {
			totalPartitions++

			if len(partition.Replicas) > len(partition.ISR) {
				underReplicated++
			}

			if partition.Leader == -1 {
				offline++
			}
		}
	}

	stats.TotalPartitions = totalPartitions
	stats.UnderReplicatedPartitions = underReplicated
	stats.OfflinePartitions = offline

	groups, err := a.client.DescribeGroups(cctx)
	if err == nil {
		stats.TotalConsumerGroups = len(groups)
	}

	return stats, nil
}

// GetBrokerDetails returns detailed information about all brokers
func (a *Admin) GetBrokerDetails(ctx context.Context) ([]domain.BrokerDetail, error) {
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	meta, err := a.client.BrokerMetadata(cctx)
	if err != nil {
		return nil, err
	}

	topics, err := a.client.ListTopics(cctx)
	if err != nil {
		return nil, err
	}

	leaderCounts := make(map[int32]int)
	for _, topic := range topics {
		if topic.IsInternal {
			continue
		}
		for _, partition := range topic.Partitions {
			if partition.Leader >= 0 {
				leaderCounts[partition.Leader]++
			}
		}
	}

	brokers := make([]domain.BrokerDetail, 0, len(meta.Brokers))
	for _, b := range meta.Brokers {
		rack := ""
		if b.Rack != nil {
			rack = *b.Rack
		}
		brokers = append(brokers, domain.BrokerDetail{
			ID:               b.NodeID,
			Host:             b.Host,
			Port:             b.Port,
			Rack:             rack,
			IsController:     b.NodeID == meta.Controller,
			LeaderPartitions: leaderCounts[b.NodeID],
		})
	}

	return brokers, nil
}

// ListConsumerGroups returns a list of consumer groups with basic info
func (a *Admin) ListConsumerGroups(ctx context.Context) ([]domain.ConsumerGroupSummary, error) {
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	groups, err := a.client.DescribeGroups(cctx)
	if err != nil {
		return nil, err
	}

	result := make([]domain.ConsumerGroupSummary, 0, len(groups))
	for groupID, group := range groups {
		result = append(result, domain.ConsumerGroupSummary{
			GroupID: groupID,
			State:   group.State,
			Members: len(group.Members),
		})
	}

	return result, nil
}
