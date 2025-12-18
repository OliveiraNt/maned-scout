package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/OliveiraNt/kdash/internal/config"
	"github.com/OliveiraNt/kdash/internal/registry"

	"github.com/go-chi/chi/v5"
)

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
	name := chi.URLParam(r, "clusterName")
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
	name := chi.URLParam(r, "clusterName")
	if err := s.clusterService.DeleteCluster(name); err != nil {
		registry.Logger.Error("api delete cluster failed", "cluster", name, "err", err)
		http.Error(w, err.Error(), 404)
		return
	}
	registry.Logger.Info("cluster deleted", "cluster", name)
	w.WriteHeader(204)
}
