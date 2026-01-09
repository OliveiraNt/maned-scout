// Package application provides the business logic layer for the Maned Scout Kafka management tool.
// It contains service implementations that orchestrate domain operations, coordinate with repositories,
// and enforce business rules for cluster management, topic operations, and consumer group monitoring.
package application

import (
	"github.com/OliveiraNt/maned-scout/internal/config"
	"github.com/OliveiraNt/maned-scout/internal/domain"
	"github.com/OliveiraNt/maned-scout/internal/utils"
)

// ClusterService provides operations related to cluster management.
type ClusterService struct {
	repo domain.ClusterRepository
}

// NewClusterService creates a new cluster service.
func NewClusterService(repo domain.ClusterRepository) *ClusterService {
	return &ClusterService{repo: repo}
}

func (s *ClusterService) getRepo() domain.ClusterRepository {
	return s.repo
}

// ListClusters lists all clusters.
func (s *ClusterService) ListClusters() []config.ClusterConfig {
	return s.repo.FindAll()
}

// GetCluster retrieves a cluster configuration by name.
func (s *ClusterService) GetCluster(name string) (config.ClusterConfig, bool) {
	return s.repo.FindByName(name)
}

// AddCluster adds a new cluster configuration.
func (s *ClusterService) AddCluster(cfg config.ClusterConfig) error {
	if cfg.Name == "" || len(cfg.Brokers) == 0 {
		return ErrInvalidClusterConfig
	}
	return s.repo.Save(cfg)
}

// UpdateCluster updates an existing cluster configuration.
func (s *ClusterService) UpdateCluster(name string, cfg config.ClusterConfig) error {
	cfg.Name = name
	return s.repo.Save(cfg)
}

// DeleteCluster removes a cluster configuration.
func (s *ClusterService) DeleteCluster(name string) error {
	return s.repo.Delete(name)
}

// GetClusterInfo retrieves basic cluster info with status and stats using existing client.
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
		if certInfo, err := cfg.GetCertificateInfo(); err == nil {
			cluster.CertInfo = certInfo
		} else {
			utils.Logger.Warn("get certificate info failed", "cluster", cfg.Name, "err", err)
		}
	}

	client, ok := s.repo.GetClient(name)
	if !ok {
		utils.Logger.Warn("get cluster info client not found", "cluster", name)
		return cluster, nil, nil
	}

	cluster.IsOnline = client.IsHealthy()
	var stats *domain.ClusterStats
	if cluster.IsOnline {
		st, err := client.GetClusterStats()
		if err != nil {
			utils.Logger.Error("get cluster stats failed", "cluster", name, "err", err)
		} else {
			stats = st
		}
	}
	return cluster, stats, nil
}

// GetClusterDetail retrieves detailed cluster information using existing client.
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
		if certInfo, err := cfg.GetCertificateInfo(); err == nil {
			cluster.CertInfo = certInfo
		} else {
			utils.Logger.Warn("get certificate info failed", "cluster", cfg.Name, "err", err)
		}
	}

	topics := make(map[string]int)
	var stats *domain.ClusterStats
	var brokerDetails []domain.BrokerDetail
	var consumerGroups []domain.ConsumerGroupSummary

	client, ok := s.repo.GetClient(name)
	if !ok {
		utils.Logger.Warn("get cluster detail client not found", "cluster", name)
		return cluster, topics, stats, brokerDetails, consumerGroups, nil
	}

	cluster.IsOnline = client.IsHealthy()
	if cluster.IsOnline {
		if tl, err := client.ListTopics(false); err == nil {
			topics = tl
		} else {
			utils.Logger.Error("list topics failed", "cluster", name, "err", err)
		}
		if st, err := client.GetClusterStats(); err == nil {
			stats = st
		} else {
			utils.Logger.Error("get cluster stats failed", "cluster", name, "err", err)
		}
		if br, err := client.GetBrokerDetails(); err == nil {
			brokerDetails = br
		} else {
			utils.Logger.Error("get broker details failed", "cluster", name, "err", err)
		}
		if cg, err := client.ListConsumerGroups(); err == nil {
			consumerGroups = cg
		} else {
			utils.Logger.Error("list consumer groups failed", "cluster", name, "err", err)
		}
	}

	return cluster, topics, stats, brokerDetails, consumerGroups, nil
}
