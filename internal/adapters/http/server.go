package httpserver

import (
	"encoding/json"
	"log"
	"net/http"

	pages "github.com/OliveiraNt/kdash/internal/adapters/http/ui/templates/pages"
	"github.com/OliveiraNt/kdash/internal/config"
	"github.com/OliveiraNt/kdash/internal/core"
	"github.com/OliveiraNt/kdash/internal/registry"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	reg        *registry.Registry
	configPath string
}

func New(reg *registry.Registry, configPath string) *Server {
	return &Server{reg: reg, configPath: configPath}
}

func (s *Server) Run(addr string) error {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	// Web UI routes
	r.Get("/", s.uiHome)
	r.Get("/cluster/{name}", s.uiClusterDetail)
	r.Get("/cluster/{name}/topics", s.uiTopicsList)

	// Cluster APIS
	r.Get("/api/clusters", s.apiListClusters)
	r.Post("/api/clusters", s.apiAddCluster)
	r.Put("/api/clusters/{name}", s.apiUpdateCluster)
	r.Delete("/api/clusters/{name}", s.apiDeleteCluster)

	// Topic Apis
	r.Get("/api/cluster/{name}/topics", s.apiListTopics)

	return http.ListenAndServe(addr, r)
}

// uiHome renders the homepage listing clusters using templ + htmx + tailwind layout.
func (s *Server) uiHome(w http.ResponseWriter, r *http.Request) {
	cfgs := s.reg.ListClusters()
	// Map config clusters to core.Cluster for UI components with stats
	clustersList := make([]pages.ClusterWithStats, 0, len(cfgs))
	for _, c := range cfgs {
		isOnline := false
		var stats *core.ClusterStats

		// Check if cluster is online
		if client, ok := s.reg.GetClient(c.Name); ok {
			isOnline = client.IsHealthy()
			if isOnline {
				// Get quick stats for the card
				clusterStats, err := client.GetClusterStats()
				if err != nil {
					log.Printf("failed to get stats for cluster %s: %v", c.Name, err)
				} else {
					stats = clusterStats
				}
			}
		}

		clustersList = append(clustersList, pages.ClusterWithStats{
			Cluster: core.Cluster{
				ID:       c.Name,
				Name:     c.Name,
				Brokers:  c.Brokers,
				IsOnline: isOnline,
			},
			Stats: stats,
		})
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.ClusterList(clustersList).Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// uiClusterDetail renders the cluster detail page with topics list
func (s *Server) uiClusterDetail(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	// Get cluster config
	cfg, ok := s.reg.GetCluster(name)
	if !ok {
		http.Error(w, "cluster not found", http.StatusNotFound)
		return
	}

	isOnline := false
	cluster := core.Cluster{ID: cfg.Name, Name: cfg.Name, Brokers: cfg.Brokers}

	// Get topics for this cluster
	topics := make(map[string]int)
	var stats *core.ClusterStats
	var brokerDetails []core.BrokerDetail
	var consumerGroups []core.ConsumerGroupSummary

	client, ok := s.reg.GetClient(name)
	if ok {
		isOnline = client.IsHealthy()
		cluster.IsOnline = isOnline

		if isOnline {
			// Get topics
			topicList, err := client.ListTopics(false)
			if err != nil {
				log.Printf("failed to list topics for cluster %s: %v", name, err)
			} else {
				topics = topicList
			}

			// Get cluster statistics
			clusterStats, err := client.GetClusterStats()
			if err != nil {
				log.Printf("failed to get cluster stats for %s: %v", name, err)
			} else {
				stats = clusterStats
			}

			// Get broker details
			brokers, err := client.GetBrokerDetails()
			if err != nil {
				log.Printf("failed to get broker details for %s: %v", name, err)
			} else {
				brokerDetails = brokers
			}

			// Get consumer groups
			groups, err := client.ListConsumerGroups()
			if err != nil {
				log.Printf("failed to list consumer groups for %s: %v", name, err)
			} else {
				consumerGroups = groups
			}
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.ClusterDetail(cluster, topics, stats, brokerDetails, consumerGroups).Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// uiTopicsList renders the topics list page for a cluster
func (s *Server) uiTopicsList(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	// Get cluster config
	_, ok := s.reg.GetCluster(name)
	if !ok {
		http.Error(w, "cluster not found", http.StatusNotFound)
		return
	}

	// Check if showInternal parameter is set (default: false)
	showInternal := r.URL.Query().Get("showInternal") == "true"

	// Get topics for this cluster
	topics := make(map[string]int)
	client, ok := s.reg.GetClient(name)
	if ok {
		topicList, err := client.ListTopics(showInternal)
		if err != nil {
			log.Printf("failed to list topics for cluster %s: %v", name, err)
		} else {
			topics = topicList
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.TopicsList(name, topics, showInternal).Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) apiListClusters(w http.ResponseWriter, r *http.Request) {
	clusters := s.reg.ListClusters()
	json.NewEncoder(w).Encode(clusters)
}

func (s *Server) apiAddCluster(w http.ResponseWriter, r *http.Request) {
	var c config.ClusterConfig
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if c.Name == "" || len(c.Brokers) == 0 {
		http.Error(w, "invalid payload", 400)
		return
	}
	if err := s.reg.AddOrUpdateCluster(c); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	// persist
	if err := s.reg.WriteToFile(s.configPath); err != nil {
		log.Printf("failed to persist config: %v", err)
	}
	w.WriteHeader(201)
}

func (s *Server) apiUpdateCluster(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var c config.ClusterConfig
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if c.Name == "" {
		c.Name = name
	}
	if err := s.reg.AddOrUpdateCluster(c); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if err := s.reg.WriteToFile(s.configPath); err != nil {
		log.Printf("failed to persist config: %v", err)
	}
	w.WriteHeader(204)
}

func (s *Server) apiDeleteCluster(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := s.reg.DeleteCluster(name); err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	if err := s.reg.WriteToFile(s.configPath); err != nil {
		log.Printf("failed to persist config: %v", err)
	}
	w.WriteHeader(204)
}

func (s *Server) apiListTopics(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	client, ok := s.reg.GetClient(name)
	if !ok {
		http.Error(w, "cluster not found", 404)
		return
	}
	topics, err := client.ListTopics(true)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(topics)
}
