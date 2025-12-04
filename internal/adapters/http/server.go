package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/OliveiraNt/kdash/internal/adapters/http/ui/templates/pages"
	"github.com/OliveiraNt/kdash/internal/application"
	"github.com/OliveiraNt/kdash/internal/config"
	"github.com/OliveiraNt/kdash/internal/domain"
	"github.com/OliveiraNt/kdash/internal/infrastructure/repository"
	"github.com/OliveiraNt/kdash/internal/registry"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	clusterService *application.ClusterService
	repo           *repository.ClusterRepository
}

func New(clusterService *application.ClusterService, repo *repository.ClusterRepository) *Server {
	return &Server{
		clusterService: clusterService,
		repo:           repo,
	}
}

func (s *Server) Run(addr string) error {
	r := chi.NewRouter()
	// replace default logger with simple request log using our logger
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			next.ServeHTTP(ww, r)
			dur := time.Since(start)
			registry.Logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration", dur.String(),
			)
		})
	})
	// Web UI routes
	r.Get("/", s.uiHome)
	r.Get("/cluster/{name}", s.uiClusterDetail)
	r.Get("/cluster/{name}/topics", s.uiTopicsList)
	r.Get("/cluster/{name}/topics/{topic}", s.uiTopicDetail)

	// Cluster APIS
	r.Get("/api/clusters", s.apiListClusters)
	r.Post("/api/clusters", s.apiAddCluster)
	r.Put("/api/clusters/{name}", s.apiUpdateCluster)
	r.Delete("/api/clusters/{name}", s.apiDeleteCluster)

	// Topic Apis
	r.Get("/api/cluster/{name}/topics", s.apiListTopics)
	r.Get("/api/cluster/{name}/topics/{topic}", s.apiGetTopicDetail)
	r.Post("/api/cluster/{name}/topics", s.apiCreateTopic)
	r.Delete("/api/cluster/{name}/topics/{topic}", s.apiDeleteTopic)
	r.Put("/api/cluster/{name}/topics/{topic}/config", s.apiUpdateTopicConfig)
	r.Post("/api/cluster/{name}/topics/{topic}/partitions", s.apiIncreasePartitions)
	r.Get("/api/cluster/{name}/topics/{topic}", s.apiGetTopicDetail)
	r.Post("/api/cluster/{name}/topics", s.apiCreateTopic)
	r.Delete("/api/cluster/{name}/topics/{topic}", s.apiDeleteTopic)
	r.Put("/api/cluster/{name}/topics/{topic}/config", s.apiUpdateTopicConfig)
	r.Post("/api/cluster/{name}/topics/{topic}/partitions", s.apiIncreasePartitions)

	registry.Logger.Info("HTTP server listening", "addr", addr)
	return http.ListenAndServe(addr, r)
}

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

// uiTopicsList renders the topics list page for a cluster
func (s *Server) uiTopicsList(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	registry.Logger.Debug("render topics list", "cluster", name)

	// Get cluster config
	_, ok := s.clusterService.GetCluster(name)
	if !ok {
		http.Error(w, "cluster not found", http.StatusNotFound)
		return
	}

	// Check if showInternal parameter is set (default: false)
	showInternal := r.URL.Query().Get("showInternal") == "true"

	// Get topics for this cluster
	topics := make(map[string]int)
	client, ok := s.repo.GetClient(name)
	if ok {
		topicList, err := client.ListTopics(showInternal)
		if err != nil {
			registry.Logger.Error("list topics failed", "cluster", name, "err", err)
		} else {
			topics = topicList
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.TopicsList(name, topics, showInternal).Render(r.Context(), w); err != nil {
		registry.Logger.Error("render topics list failed", "cluster", name, "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// uiTopicDetail renders the topic detail page
func (s *Server) uiTopicDetail(w http.ResponseWriter, r *http.Request) {
	clusterName := chi.URLParam(r, "name")
	topicName := chi.URLParam(r, "topic")
	registry.Logger.Debug("render topic detail", "cluster", clusterName, "topic", topicName)

	// Get cluster config
	_, ok := s.clusterService.GetCluster(clusterName)
	if !ok {
		http.Error(w, "cluster not found", http.StatusNotFound)
		return
	}

	// Get topic detail
	var topicDetail *domain.TopicDetail
	client, ok := s.repo.GetClient(clusterName)
	if ok {
		detail, err := client.GetTopicDetail(topicName)
		if err != nil {
			registry.Logger.Error("get topic detail failed", "cluster", clusterName, "topic", topicName, "err", err)
			http.Error(w, "topic not found", http.StatusNotFound)
			return
		}
		topicDetail = detail
	} else {
		http.Error(w, "cluster not available", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.TopicDetail(clusterName, topicDetail).Render(r.Context(), w); err != nil {
		registry.Logger.Error("render topic detail failed", "cluster", clusterName, "topic", topicName, "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// APIs
func (s *Server) apiListClusters(w http.ResponseWriter, r *http.Request) {
	_ = r
	clusters := s.clusterService.ListClusters()
	registry.Logger.Debug("api list clusters", "count", len(clusters))
	if err := json.NewEncoder(w).Encode(clusters); err != nil {
		registry.Logger.Error("encode clusters failed", "err", err)
	}
}

func (s *Server) apiAddCluster(w http.ResponseWriter, r *http.Request) {
	var c config.ClusterConfig
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		registry.Logger.Warn("api add cluster bad request", "err", err)
		http.Error(w, err.Error(), 400)
		return
	}
	if err := s.clusterService.AddCluster(c); err != nil {
		registry.Logger.Error("api add cluster failed", "cluster", c.Name, "err", err)
		http.Error(w, err.Error(), 500)
		return
	}
	registry.Logger.Info("cluster added", "cluster", c.Name)
	w.WriteHeader(201)
}

func (s *Server) apiUpdateCluster(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var c config.ClusterConfig
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		registry.Logger.Warn("api update cluster bad request", "cluster", name, "err", err)
		http.Error(w, err.Error(), 400)
		return
	}
	if err := s.clusterService.UpdateCluster(name, c); err != nil {
		registry.Logger.Error("api update cluster failed", "cluster", name, "err", err)
		http.Error(w, err.Error(), 500)
		return
	}
	registry.Logger.Info("cluster updated", "cluster", name)
	w.WriteHeader(204)
}

func (s *Server) apiDeleteCluster(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := s.clusterService.DeleteCluster(name); err != nil {
		registry.Logger.Error("api delete cluster failed", "cluster", name, "err", err)
		http.Error(w, err.Error(), 404)
		return
	}
	registry.Logger.Info("cluster deleted", "cluster", name)
	w.WriteHeader(204)
}

func (s *Server) apiListTopics(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	client, ok := s.repo.GetClient(name)
	if !ok {
		registry.Logger.Warn("api list topics cluster not found", "cluster", name)
		http.Error(w, "cluster not found", 404)
		return
	}
	topics, err := client.ListTopics(true)
	if err != nil {
		registry.Logger.Error("api list topics failed", "cluster", name, "err", err)
		http.Error(w, err.Error(), 500)
		return
	}
	if err := json.NewEncoder(w).Encode(topics); err != nil {
		registry.Logger.Error("encode topics failed", "cluster", name, "err", err)
	}
}

func (s *Server) apiGetTopicDetail(w http.ResponseWriter, r *http.Request) {
	clusterName := chi.URLParam(r, "name")
	topicName := chi.URLParam(r, "topic")

	client, ok := s.repo.GetClient(clusterName)
	if !ok {
		registry.Logger.Warn("api get topic detail cluster not found", "cluster", clusterName)
		http.Error(w, "cluster not found", 404)
		return
	}

	topicDetail, err := client.GetTopicDetail(topicName)
	if err != nil {
		registry.Logger.Error("api get topic detail failed", "cluster", clusterName, "topic", topicName, "err", err)
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(topicDetail); err != nil {
		registry.Logger.Error("encode topic detail failed", "cluster", clusterName, "topic", topicName, "err", err)
	}
}

func (s *Server) apiCreateTopic(w http.ResponseWriter, r *http.Request) {
	clusterName := chi.URLParam(r, "name")

	var req domain.CreateTopicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		registry.Logger.Warn("api create topic bad request", "cluster", clusterName, "err", err)
		http.Error(w, err.Error(), 400)
		return
	}

	// Validate request
	if req.Name == "" {
		http.Error(w, "topic name is required", 400)
		return
	}
	if req.NumPartitions <= 0 {
		http.Error(w, "number of partitions must be greater than 0", 400)
		return
	}
	if req.ReplicationFactor <= 0 {
		http.Error(w, "replication factor must be greater than 0", 400)
		return
	}

	client, ok := s.repo.GetClient(clusterName)
	if !ok {
		registry.Logger.Warn("api create topic cluster not found", "cluster", clusterName)
		http.Error(w, "cluster not found", 404)
		return
	}

	if err := client.CreateTopic(req); err != nil {
		registry.Logger.Error("api create topic failed", "cluster", clusterName, "topic", req.Name, "err", err)
		http.Error(w, err.Error(), 500)
		return
	}

	registry.Logger.Info("topic created", "cluster", clusterName, "topic", req.Name)
	w.WriteHeader(201)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "topic created successfully"}); err != nil {
		registry.Logger.Error("encode response failed", "err", err)
	}
}

func (s *Server) apiDeleteTopic(w http.ResponseWriter, r *http.Request) {
	clusterName := chi.URLParam(r, "name")
	topicName := chi.URLParam(r, "topic")

	client, ok := s.repo.GetClient(clusterName)
	if !ok {
		registry.Logger.Warn("api delete topic cluster not found", "cluster", clusterName)
		http.Error(w, "cluster not found", 404)
		return
	}

	if err := client.DeleteTopic(topicName); err != nil {
		registry.Logger.Error("api delete topic failed", "cluster", clusterName, "topic", topicName, "err", err)
		http.Error(w, err.Error(), 500)
		return
	}

	registry.Logger.Info("topic deleted", "cluster", clusterName, "topic", topicName)
	w.WriteHeader(204)
}

func (s *Server) apiUpdateTopicConfig(w http.ResponseWriter, r *http.Request) {
	clusterName := chi.URLParam(r, "name")
	topicName := chi.URLParam(r, "topic")

	var req domain.UpdateTopicConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		registry.Logger.Warn("api update topic config bad request", "cluster", clusterName, "topic", topicName, "err", err)
		http.Error(w, err.Error(), 400)
		return
	}

	if len(req.Configs) == 0 {
		http.Error(w, "configs are required", 400)
		return
	}

	client, ok := s.repo.GetClient(clusterName)
	if !ok {
		registry.Logger.Warn("api update topic config cluster not found", "cluster", clusterName)
		http.Error(w, "cluster not found", 404)
		return
	}

	if err := client.UpdateTopicConfig(topicName, req); err != nil {
		registry.Logger.Error("api update topic config failed", "cluster", clusterName, "topic", topicName, "err", err)
		http.Error(w, err.Error(), 500)
		return
	}

	registry.Logger.Info("topic config updated", "cluster", clusterName, "topic", topicName)
	w.WriteHeader(200)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "topic config updated successfully"}); err != nil {
		registry.Logger.Error("encode response failed", "err", err)
	}
}

func (s *Server) apiIncreasePartitions(w http.ResponseWriter, r *http.Request) {
	clusterName := chi.URLParam(r, "name")
	topicName := chi.URLParam(r, "topic")

	var req domain.IncreasePartitionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		registry.Logger.Warn("api increase partitions bad request", "cluster", clusterName, "topic", topicName, "err", err)
		http.Error(w, err.Error(), 400)
		return
	}

	if req.TotalPartitions <= 0 {
		http.Error(w, "total partitions must be greater than 0", 400)
		return
	}

	client, ok := s.repo.GetClient(clusterName)
	if !ok {
		registry.Logger.Warn("api increase partitions cluster not found", "cluster", clusterName)
		http.Error(w, "cluster not found", 404)
		return
	}

	if err := client.IncreasePartitions(topicName, req); err != nil {
		registry.Logger.Error("api increase partitions failed", "cluster", clusterName, "topic", topicName, "err", err)
		http.Error(w, err.Error(), 500)
		return
	}

	registry.Logger.Info("topic partitions increased", "cluster", clusterName, "topic", topicName, "partitions", req.TotalPartitions)
	w.WriteHeader(200)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "partitions increased successfully"}); err != nil {
		registry.Logger.Error("encode response failed", "err", err)
	}
}
