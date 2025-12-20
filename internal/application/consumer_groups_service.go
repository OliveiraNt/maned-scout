package application

import (
	"context"

	"github.com/OliveiraNt/maned-scout/internal/domain"
	"github.com/OliveiraNt/maned-scout/internal/utils"
	"github.com/twmb/franz-go/pkg/kadm"
)

type ConsumerGroupsService struct {
	clusterService *ClusterService
	repo           domain.ClusterRepository
}

func NewConsumerGroupsService(clusterService *ClusterService) *ConsumerGroupsService {
	return &ConsumerGroupsService{
		clusterService: clusterService,
		repo:           clusterService.getRepo(),
	}
}

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

func (s *ConsumerGroupsService) GetTopicsLags(group kadm.GroupLag) kadm.GroupTopicsLag {
	return group.TotalByTopic()
}
