package application

import (
	"context"

	"github.com/OliveiraNt/kdash/internal/domain"
	"github.com/OliveiraNt/kdash/internal/registry"
)

// TopicService handles topic-related business operations.
type TopicService struct {
	clusterService *ClusterService
	factory        ClientFactory
}

// NewTopicService creates a new topic service.
func NewTopicService(clusterService *ClusterService, factory ClientFactory) *TopicService {
	return &TopicService{
		clusterService: clusterService,
		factory:        factory,
	}
}

// ListTopics retrieves all topics from a cluster.
func (s *TopicService) ListTopics(clusterName string, showInternal bool) (map[string]int, error) {
	cfg, ok := s.clusterService.GetCluster(clusterName)
	if !ok {
		return nil, ErrClusterNotFound
	}

	client, err := s.factory.CreateClient(cfg)
	if err != nil {
		registry.Logger.Error("create client failed", "cluster", clusterName, "err", err)
		return nil, err
	}
	defer client.Close()

	topics, err := client.ListTopics(showInternal)
	if err != nil {
		registry.Logger.Error("list topics failed", "cluster", clusterName, "err", err)
		return nil, err
	}

	return topics, nil
}

// GetTopicDetail retrieves detailed information about a specific topic.
func (s *TopicService) GetTopicDetail(clusterName, topicName string) (*domain.TopicDetail, error) {
	cfg, ok := s.clusterService.GetCluster(clusterName)
	if !ok {
		return nil, ErrClusterNotFound
	}

	client, err := s.factory.CreateClient(cfg)
	if err != nil {
		registry.Logger.Error("create client failed", "cluster", clusterName, "err", err)
		return nil, err
	}
	defer client.Close()

	detail, err := client.GetTopicDetail(topicName)
	if err != nil {
		registry.Logger.Error("get topic detail failed", "cluster", clusterName, "topic", topicName, "err", err)
		return nil, err
	}

	return detail, nil
}

// CreateTopic creates a new topic in the cluster.
func (s *TopicService) CreateTopic(clusterName string, req domain.CreateTopicRequest) error {
	if req.Name == "" {
		return ErrInvalidTopicName
	}
	if req.NumPartitions <= 0 {
		return ErrInvalidPartitionCount
	}
	if req.ReplicationFactor <= 0 {
		return ErrInvalidReplicationFactor
	}

	cfg, ok := s.clusterService.GetCluster(clusterName)
	if !ok {
		return ErrClusterNotFound
	}

	client, err := s.factory.CreateClient(cfg)
	if err != nil {
		registry.Logger.Error("create client failed", "cluster", clusterName, "err", err)
		return err
	}
	defer client.Close()

	if err := client.CreateTopic(req); err != nil {
		registry.Logger.Error("create topic failed", "cluster", clusterName, "topic", req.Name, "err", err)
		return err
	}

	registry.Logger.Info("topic created", "cluster", clusterName, "topic", req.Name)
	return nil
}

// DeleteTopic removes a topic from the cluster.
func (s *TopicService) DeleteTopic(clusterName, topicName string) error {
	cfg, ok := s.clusterService.GetCluster(clusterName)
	if !ok {
		return ErrClusterNotFound
	}

	client, err := s.factory.CreateClient(cfg)
	if err != nil {
		registry.Logger.Error("create client failed", "cluster", clusterName, "err", err)
		return err
	}
	defer client.Close()

	if err := client.DeleteTopic(topicName); err != nil {
		registry.Logger.Error("delete topic failed", "cluster", clusterName, "topic", topicName, "err", err)
		return err
	}

	registry.Logger.Info("topic deleted", "cluster", clusterName, "topic", topicName)
	return nil
}

// UpdateTopicConfig updates the configuration of an existing topic.
func (s *TopicService) UpdateTopicConfig(clusterName, topicName string, req domain.UpdateTopicConfigRequest) error {
	if len(req.Configs) == 0 {
		return ErrInvalidTopicConfig
	}

	cfg, ok := s.clusterService.GetCluster(clusterName)
	if !ok {
		return ErrClusterNotFound
	}

	client, err := s.factory.CreateClient(cfg)
	if err != nil {
		registry.Logger.Error("create client failed", "cluster", clusterName, "err", err)
		return err
	}
	defer client.Close()

	if err := client.UpdateTopicConfig(topicName, req); err != nil {
		registry.Logger.Error("update topic config failed", "cluster", clusterName, "topic", topicName, "err", err)
		return err
	}

	registry.Logger.Info("topic config updated", "cluster", clusterName, "topic", topicName)
	return nil
}

// IncreasePartitions increases the number of partitions for a topic.
func (s *TopicService) IncreasePartitions(clusterName, topicName string, req domain.IncreasePartitionsRequest) error {
	if req.TotalPartitions <= 0 {
		return ErrInvalidPartitionCount
	}

	cfg, ok := s.clusterService.GetCluster(clusterName)
	if !ok {
		return ErrClusterNotFound
	}

	client, err := s.factory.CreateClient(cfg)
	if err != nil {
		registry.Logger.Error("create client failed", "cluster", clusterName, "err", err)
		return err
	}
	defer client.Close()

	if err := client.IncreasePartitions(topicName, req); err != nil {
		registry.Logger.Error("increase partitions failed", "cluster", clusterName, "topic", topicName, "err", err)
		return err
	}

	registry.Logger.Info("topic partitions increased", "cluster", clusterName, "topic", topicName, "partitions", req.TotalPartitions)
	return nil
}

// StreamMessages streams messages from a topic to a channel.
func (s *TopicService) StreamMessages(ctx context.Context, clusterName, topicName string, out chan<- domain.Message) error {
	cfg, ok := s.clusterService.GetCluster(clusterName)
	if !ok {
		return ErrClusterNotFound
	}

	client, err := s.factory.CreateClient(cfg)
	if err != nil {
		registry.Logger.Error("create client failed", "cluster", clusterName, "err", err)
		return err
	}

	// Note: client.Close() is deferred to the caller context cancellation
	go func() {
		<-ctx.Done()
		client.Close()
	}()

	client.StreamMessages(ctx, topicName, out)
	return nil
}
