package application

import (
	"github.com/OliveiraNt/kdash/internal/config"
	"github.com/OliveiraNt/kdash/internal/domain"
	"github.com/OliveiraNt/kdash/internal/registry"
)

// ClusterService handles cluster-related business operations.
type ClusterService struct {
	repo    domain.ClusterRepository
	factory ClientFactory
}

// ClientFactory creates Kafka clients from configuration.
type ClientFactory interface {
	CreateClient(cfg config.ClusterConfig) (domain.KafkaClient, error)
}

// NewClusterService creates a new cluster service.
func NewClusterService(repo domain.ClusterRepository, factory ClientFactory) *ClusterService {
	return &ClusterService{
		repo:    repo,
		factory: factory,
	}
}

// ListClusters returns all cluster configurations.
func (s *ClusterService) ListClusters() []config.ClusterConfig {
	return s.repo.FindAll()
}

// GetCluster retrieves a specific cluster configuration.
func (s *ClusterService) GetCluster(name string) (config.ClusterConfig, bool) {
	return s.repo.FindByName(name)
}

// AddCluster adds a new cluster or updates an existing one.
func (s *ClusterService) AddCluster(cfg config.ClusterConfig) error {
	if cfg.Name == "" || len(cfg.Brokers) == 0 {
		return ErrInvalidClusterConfig
	}
	return s.repo.Save(cfg)
}

// UpdateCluster updates an existing cluster.
func (s *ClusterService) UpdateCluster(name string, cfg config.ClusterConfig) error {
	if cfg.Name == "" {
		cfg.Name = name
	}
	return s.repo.Save(cfg)
}

// DeleteCluster removes a cluster.
func (s *ClusterService) DeleteCluster(name string) error {
	return s.repo.Delete(name)
}

// GetClusterWithStats retrieves cluster information with statistics.
func (s *ClusterService) GetClusterWithStats(name string, client domain.KafkaClient) (*domain.Cluster, *domain.ClusterStats, error) {
	cfg, ok := s.repo.FindByName(name)
	if !ok {
		return nil, nil, ErrClusterNotFound
	}

	cluster := &domain.Cluster{
		ID:       cfg.Name,
		Name:     cfg.Name,
		Brokers:  cfg.Brokers,
		AuthType: cfg.GetAuthType(),
	}

	if cfg.HasCertificate() {
		certInfo, err := cfg.GetCertificateInfo()
		if err != nil {
			registry.Logger.Warn("get certificate info failed", "cluster", cfg.Name, "err", err)
		} else {
			cluster.CertInfo = certInfo
		}
	}

	var stats *domain.ClusterStats
	if client != nil {
		cluster.IsOnline = client.IsHealthy()
		if cluster.IsOnline {
			var err error
			stats, err = client.GetClusterStats()
			if err != nil {
				registry.Logger.Error("get stats failed", "cluster", name, "err", err)
			}
		}
	}

	return cluster, stats, nil
}

// GetClusterInfo retrieves basic cluster information with online status and statistics.
func (s *ClusterService) GetClusterInfo(name string) (*domain.Cluster, *domain.ClusterStats, error) {
	cfg, ok := s.repo.FindByName(name)
	if !ok {
		return nil, nil, ErrClusterNotFound
	}

	cluster := &domain.Cluster{
		ID:       cfg.Name,
		Name:     cfg.Name,
		Brokers:  cfg.Brokers,
		AuthType: cfg.GetAuthType(),
	}

	if cfg.HasCertificate() {
		certInfo, err := cfg.GetCertificateInfo()
		if err != nil {
			registry.Logger.Warn("get certificate info failed", "cluster", cfg.Name, "err", err)
		} else {
			cluster.CertInfo = certInfo
		}
	}

	client, err := s.factory.CreateClient(cfg)
	if err != nil {
		registry.Logger.Error("create client failed", "cluster", name, "err", err)
		return cluster, nil, nil
	}
	defer client.Close()

	cluster.IsOnline = client.IsHealthy()
	var stats *domain.ClusterStats
	if cluster.IsOnline {
		stats, err = client.GetClusterStats()
		if err != nil {
			registry.Logger.Error("get cluster stats failed", "cluster", name, "err", err)
		}
	}

	return cluster, stats, nil
}

// GetClusterDetail retrieves detailed cluster information including topics, brokers, and consumer groups.
func (s *ClusterService) GetClusterDetail(name string) (*domain.Cluster, map[string]int, *domain.ClusterStats, []domain.BrokerDetail, []domain.ConsumerGroupSummary, error) {
	cfg, ok := s.repo.FindByName(name)
	if !ok {
		return nil, nil, nil, nil, nil, ErrClusterNotFound
	}

	cluster := &domain.Cluster{
		ID:       cfg.Name,
		Name:     cfg.Name,
		Brokers:  cfg.Brokers,
		AuthType: cfg.GetAuthType(),
	}

	if cfg.HasCertificate() {
		certInfo, err := cfg.GetCertificateInfo()
		if err != nil {
			registry.Logger.Warn("get certificate info failed", "cluster", cfg.Name, "err", err)
		} else {
			cluster.CertInfo = certInfo
		}
	}

	topics := make(map[string]int)
	var stats *domain.ClusterStats
	var brokerDetails []domain.BrokerDetail
	var consumerGroups []domain.ConsumerGroupSummary

	client, err := s.factory.CreateClient(cfg)
	if err != nil {
		registry.Logger.Error("create client failed", "cluster", name, "err", err)
		return cluster, topics, stats, brokerDetails, consumerGroups, nil
	}
	defer client.Close()

	cluster.IsOnline = client.IsHealthy()
	if cluster.IsOnline {
		topicList, err := client.ListTopics(false)
		if err != nil {
			registry.Logger.Error("list topics failed", "cluster", name, "err", err)
		} else {
			topics = topicList
		}

		stats, err = client.GetClusterStats()
		if err != nil {
			registry.Logger.Error("get cluster stats failed", "cluster", name, "err", err)
		}

		brokerDetails, err = client.GetBrokerDetails()
		if err != nil {
			registry.Logger.Error("get broker details failed", "cluster", name, "err", err)
		}

		consumerGroups, err = client.ListConsumerGroups()
		if err != nil {
			registry.Logger.Error("list consumer groups failed", "cluster", name, "err", err)
		}
	}

	return cluster, topics, stats, brokerDetails, consumerGroups, nil
}
