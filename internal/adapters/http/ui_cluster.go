package httpserver

import (
	"net/http"

	"github.com/OliveiraNt/kdash/internal/adapters/http/ui/templates/pages"
	"github.com/OliveiraNt/kdash/internal/config"
	"github.com/OliveiraNt/kdash/internal/domain"
	"github.com/OliveiraNt/kdash/internal/registry"

	"github.com/go-chi/chi/v5"
)

func (s *Server) uiHome(w http.ResponseWriter, r *http.Request) {
	registry.Logger.Debug("render home")
	cfgs := s.clusterService.ListClusters()
	clustersList := make([]pages.ClusterWithStats, 0, len(cfgs))
	for _, c := range cfgs {
		isOnline := false
		var stats *domain.ClusterStats
		if client, ok := s.repo.GetClient(c.Name); ok {
			isOnline = client.IsHealthy()
			if isOnline {
				clusterStats, err := client.GetClusterStats()
				if err != nil {
					registry.Logger.Error("get cluster stats failed", "cluster", c.Name, "err", err)
				} else {
					stats = clusterStats
				}
			}
		}
		var certInfo *config.CertificateInfo
		if c.HasCertificate() {
			info, err := c.GetCertificateInfo()
			if err != nil {
				registry.Logger.Warn("get certificate info failed", "cluster", c.Name, "err", err)
			} else {
				certInfo = info
			}
		}
		clustersList = append(clustersList, pages.ClusterWithStats{
			Cluster: domain.Cluster{
				ID:       c.Name,
				Name:     c.Name,
				Brokers:  c.Brokers,
				IsOnline: isOnline,
				AuthType: c.GetAuthType(),
				CertInfo: certInfo,
			},
			Stats: stats,
		})
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.ClusterList(clustersList).Render(r.Context(), w); err != nil {
		registry.Logger.Error("render home failed", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) uiClusterDetail(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	registry.Logger.Debug("render cluster detail", "cluster", name)
	cfg, ok := s.clusterService.GetCluster(name)
	if !ok {
		http.Error(w, "cluster not found", http.StatusNotFound)
		return
	}

	isOnline := false
	cluster := domain.Cluster{
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

	client, ok := s.repo.GetClient(name)
	if ok {
		isOnline = client.IsHealthy()
		cluster.IsOnline = isOnline

		if isOnline {
			topicList, err := client.ListTopics(false)
			if err != nil {
				registry.Logger.Error("list topics failed", "cluster", name, "err", err)
			} else {
				topics = topicList
			}
			clusterStats, err := client.GetClusterStats()
			if err != nil {
				registry.Logger.Error("get cluster stats failed", "cluster", name, "err", err)
			} else {
				stats = clusterStats
			}
			brokers, err := client.GetBrokerDetails()
			if err != nil {
				registry.Logger.Error("get broker details failed", "cluster", name, "err", err)
			} else {
				brokerDetails = brokers
			}
			groups, err := client.ListConsumerGroups()
			if err != nil {
				registry.Logger.Error("list consumer groups failed", "cluster", name, "err", err)
			} else {
				consumerGroups = groups
			}
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.ClusterDetail(cluster, topics, stats, brokerDetails, consumerGroups).Render(r.Context(), w); err != nil {
		registry.Logger.Error("render cluster detail failed", "cluster", name, "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
