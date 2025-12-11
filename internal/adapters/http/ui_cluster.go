package httpserver

import (
	"net/http"

	"github.com/OliveiraNt/kdash/internal/adapters/http/ui/templates/pages"
	"github.com/OliveiraNt/kdash/internal/config"
	"github.com/OliveiraNt/kdash/internal/domain"
	"github.com/OliveiraNt/kdash/internal/registry"

	"github.com/go-chi/chi/v5"
)

// uiHome renders the homepage listing clusters using templ + htmx + tailwind layout.
func (s *Server) uiHome(w http.ResponseWriter, r *http.Request) {
	registry.Logger.Debug("render home")
	cfgs := s.clusterService.ListClusters()
	// Map config clusters to domain.Cluster for UI components with stats
	clustersList := make([]pages.ClusterWithStats, 0, len(cfgs))
	for _, c := range cfgs {
		isOnline := false
		var stats *domain.ClusterStats

		// Check if cluster is online
		if client, ok := s.repo.GetClient(c.Name); ok {
			isOnline = client.IsHealthy()
			if isOnline {
				// Get quick stats for the card
				clusterStats, err := client.GetClusterStats()
				if err != nil {
					registry.Logger.Error("get cluster stats failed", "cluster", c.Name, "err", err)
				} else {
					stats = clusterStats
				}
			}
		}

		// Get certificate info if applicable
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

// uiClusterDetail renders the cluster detail page with topics list
func (s *Server) uiClusterDetail(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	registry.Logger.Debug("render cluster detail", "cluster", name)

	// Get cluster config
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

	// Get certificate info if applicable
	if cfg.HasCertificate() {
		certInfo, err := cfg.GetCertificateInfo()
		if err != nil {
			registry.Logger.Warn("get certificate info failed", "cluster", cfg.Name, "err", err)
		} else {
			cluster.CertInfo = certInfo
		}
	}

	// Get topics for this cluster
	topics := make(map[string]int)
	var stats *domain.ClusterStats
	var brokerDetails []domain.BrokerDetail
	var consumerGroups []domain.ConsumerGroupSummary

	client, ok := s.repo.GetClient(name)
	if ok {
		isOnline = client.IsHealthy()
		cluster.IsOnline = isOnline

		if isOnline {
			// Get topics
			topicList, err := client.ListTopics(false)
			if err != nil {
				registry.Logger.Error("list topics failed", "cluster", name, "err", err)
			} else {
				topics = topicList
			}

			// Get cluster statistics
			clusterStats, err := client.GetClusterStats()
			if err != nil {
				registry.Logger.Error("get cluster stats failed", "cluster", name, "err", err)
			} else {
				stats = clusterStats
			}

			// Get broker details
			brokers, err := client.GetBrokerDetails()
			if err != nil {
				registry.Logger.Error("get broker details failed", "cluster", name, "err", err)
			} else {
				brokerDetails = brokers
			}

			// Get consumer groups
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
