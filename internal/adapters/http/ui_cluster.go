package httpserver

import (
	"net/http"

	"github.com/OliveiraNt/maned-scout/internal/adapters/http/ui/templates/pages"
	"github.com/OliveiraNt/maned-scout/internal/utils"

	"github.com/go-chi/chi/v5"
)

func (s *Server) uiHome(w http.ResponseWriter, r *http.Request) {
	cfgs := s.clusterService.ListClusters()
	clustersList := make([]pages.ClusterWithStats, 0, len(cfgs))
	for _, c := range cfgs {
		cluster, stats, err := s.clusterService.GetClusterInfo(c.Name)
		if err != nil {
			utils.Logger.Error("get cluster info failed", "cluster", c.Name, "err", err)
			continue
		}
		clustersList = append(clustersList, pages.ClusterWithStats{
			Cluster: *cluster,
			Stats:   stats,
		})
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.ClusterList(clustersList).Render(r.Context(), w); err != nil {
		utils.Logger.Error("render home failed", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) uiClusterDetail(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "clusterName")
	utils.Logger.Debug("render cluster detail", "cluster", name)

	cluster, topics, stats, brokerDetails, consumerGroups, err := s.clusterService.GetClusterDetail(name)
	if err != nil {
		utils.Logger.Error("get cluster detail failed", "cluster", name, "err", err)
		http.Error(w, "cluster not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.ClusterDetail(*cluster, topics, stats, brokerDetails, consumerGroups).Render(r.Context(), w); err != nil {
		utils.Logger.Error("render cluster detail failed", "cluster", name, "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
