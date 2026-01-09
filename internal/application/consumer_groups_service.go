// Package application provides the application layer services that orchestrate business logic
// and coordinate operations across domain entities and infrastructure components.
// It includes services for managing clusters, consumer groups, topics, and their interactions.
package application

import (
	"context"

	"github.com/OliveiraNt/maned-scout/internal/domain"
	"github.com/OliveiraNt/maned-scout/internal/utils"
	"github.com/twmb/franz-go/pkg/kadm"
)

// ConsumerGroupsService provides operations related to consumer groups.
type ConsumerGroupsService struct {
	clusterService *ClusterService
	repo           domain.ClusterRepository
}

// NewConsumerGroupsService creates a new consumer groups service.
func NewConsumerGroupsService(clusterService *ClusterService) *ConsumerGroupsService {
	return &ConsumerGroupsService{
		clusterService: clusterService,
		repo:           clusterService.getRepo(),
	}
}

// ListConsumerGroupsWithLagFromTopic returns the lag for all consumer groups from a specific topic.
func (s *ConsumerGroupsService) ListConsumerGroupsWithLagFromTopic(ctx context.Context, clusterName, topicName string) (kadm.DescribedGroupLags, error) {
	_, ok := s.clusterService.GetCluster(clusterName)
	if !ok {
		return nil, ErrClusterNotFound
	}

	client, ok := s.repo.GetClient(clusterName)
	if !ok {
		utils.Logger.Warn("get topic detail client not found", "cluster", clusterName)
		return nil, ErrClusterNotFound
	}

	return client.ListConsumerGroupsWithLagFromTopic(ctx, nil, topicName)
}

// FetchConsumerGroupWithLag returns the lag for a specific consumer group.
func (s *ConsumerGroupsService) FetchConsumerGroupWithLag(ctx context.Context, clusterName, groupName string) (kadm.DescribedGroupLag, error) {
	_, ok := s.clusterService.GetCluster(clusterName)
	if !ok {
		return kadm.DescribedGroupLag{}, ErrClusterNotFound
	}

	client, ok := s.repo.GetClient(clusterName)
	if !ok {
		utils.Logger.Warn("get topic detail client not found", "cluster", clusterName)
		return kadm.DescribedGroupLag{}, ErrClusterNotFound
	}

	lags, err := client.ListConsumerGroupsWithLagFromTopic(ctx, []string{groupName}, "")
	if err != nil {
		return kadm.DescribedGroupLag{}, err
	}

	if lag, ok := lags[groupName]; ok {
		return lag, nil
	}

	return kadm.DescribedGroupLag{}, nil
}

// GetTopicsLags calculates and returns the total lag for each topic within a consumer group.
func (s *ConsumerGroupsService) GetTopicsLags(group kadm.GroupLag) kadm.GroupTopicsLag {
	return group.TotalByTopic()
}
