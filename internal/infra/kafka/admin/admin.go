package admin

import (
	"context"
	"strconv"
	"time"

	"github.com/OliveiraNt/kdash/internal/core"
	"github.com/twmb/franz-go/pkg/kadm"
)

// AdminClient defines the minimal interface used by admin helpers.
type AdminClient interface {
	ListTopics(ctx context.Context, topics ...string) (kadm.TopicDetails, error)
	ListTopicsWithInternal(ctx context.Context, topics ...string) (kadm.TopicDetails, error)
	BrokerMetadata(ctx context.Context) (kadm.Metadata, error)
	ListGroups(ctx context.Context, groups ...string) (kadm.DescribedGroups, error)
}

// KadmAdmin adapts *kadm.Client to AdminClient interface.
type KadmAdmin struct {
	c *kadm.Client
}

func NewKadmAdmin(c *kadm.Client) *KadmAdmin {
	return &KadmAdmin{c: c}
}

func (k *KadmAdmin) ListTopics(ctx context.Context, topics ...string) (kadm.TopicDetails, error) {
	return k.c.ListTopics(ctx, topics...)
}

func (k *KadmAdmin) ListTopicsWithInternal(ctx context.Context, topics ...string) (kadm.TopicDetails, error) {
	return k.c.ListTopicsWithInternal(ctx, topics...)
}

func (k *KadmAdmin) BrokerMetadata(ctx context.Context) (kadm.Metadata, error) {
	return k.c.BrokerMetadata(ctx)
}

func (k *KadmAdmin) ListGroups(ctx context.Context, groups ...string) (kadm.DescribedGroups, error) {
	return k.c.DescribeGroups(ctx, groups...)
}

// ListTopics returns topics as a simplified map name->partitions.
// If showInternal is true, includes internal topics; otherwise excludes them.
func ListTopics(ctx context.Context, admin AdminClient, showInternal bool) (map[string]int, error) {
	// use a reasonable timeout
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var m kadm.TopicDetails
	var err error

	if showInternal {
		m, err = admin.ListTopicsWithInternal(cctx)
	} else {
		m, err = admin.ListTopics(cctx)
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

// GetClusterInfo returns a core.Cluster with broker addresses and cluster ID.
func GetClusterInfo(ctx context.Context, admin AdminClient) (*core.Cluster, error) {
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	meta, err := admin.BrokerMetadata(cctx)
	if err != nil {
		return nil, err
	}
	brokers := make([]string, len(meta.Brokers))
	for i, b := range meta.Brokers {
		brokers[i] = b.Host + ":" + strconv.Itoa(int(b.Port))
	}
	return &core.Cluster{ID: meta.Cluster, Brokers: brokers}, nil
}

// GetClusterStats returns detailed statistics about the cluster
func GetClusterStats(ctx context.Context, admin AdminClient) (*core.ClusterStats, error) {
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	stats := &core.ClusterStats{}

	// Get topics info
	topics, err := admin.ListTopics(cctx)
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

			// Check if partition is under-replicated
			if len(partition.Replicas) > len(partition.ISR) {
				underReplicated++
			}

			// Check if partition is offline (no leader)
			if partition.Leader == -1 {
				offline++
			}
		}
	}

	stats.TotalPartitions = totalPartitions
	stats.UnderReplicatedPartitions = underReplicated
	stats.OfflinePartitions = offline

	// Get consumer groups count
	groups, err := admin.ListGroups(cctx)
	if err == nil {
		stats.TotalConsumerGroups = len(groups)
	}

	return stats, nil
}

// GetBrokerDetails returns detailed information about all brokers
func GetBrokerDetails(ctx context.Context, admin AdminClient) ([]core.BrokerDetail, error) {
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	meta, err := admin.BrokerMetadata(cctx)
	if err != nil {
		return nil, err
	}

	// Get topics to count leader partitions per broker
	topics, err := admin.ListTopics(cctx)
	if err != nil {
		return nil, err
	}

	// Count leader partitions per broker
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

	// Build broker details
	brokers := make([]core.BrokerDetail, 0, len(meta.Brokers))
	for _, b := range meta.Brokers {
		rack := ""
		if b.Rack != nil {
			rack = *b.Rack
		}
		brokers = append(brokers, core.BrokerDetail{
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
func ListConsumerGroups(ctx context.Context, admin AdminClient) ([]core.ConsumerGroupSummary, error) {
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	groups, err := admin.ListGroups(cctx)
	if err != nil {
		return nil, err
	}

	result := make([]core.ConsumerGroupSummary, 0, len(groups))
	for groupID, group := range groups {
		result = append(result, core.ConsumerGroupSummary{
			GroupID: groupID,
			State:   group.State,
			Members: len(group.Members),
		})
	}

	return result, nil
}
