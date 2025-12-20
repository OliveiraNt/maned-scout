package httpserver

import (
	"net/http"

	"github.com/OliveiraNt/maned-scout/internal/adapters/http/ui/templates/pages"
	"github.com/OliveiraNt/maned-scout/internal/application"
	"github.com/OliveiraNt/maned-scout/internal/utils"
	"github.com/go-chi/chi/v5"
)

func (s *Server) uiConsumerGroupList(w http.ResponseWriter, r *http.Request) {
	clusterName := chi.URLParam(r, "clusterName")
	utils.Logger.Debug("render consumer group list", "cluster", clusterName)
	_, ok := s.clusterService.GetCluster(clusterName)
	if !ok {
		http.Error(w, "cluster not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.ConsumerGroups(clusterName).Render(r.Context(), w); err != nil {
		utils.Logger.Error("render consumer group list view failed", "err", err)
		http.Error(w, "failed to render consumer group list view", 500)
		return
	}
}

func (s *Server) uiConsumerGroupDetail(w http.ResponseWriter, r *http.Request) {
	clusterName := chi.URLParam(r, "clusterName")
	groupName := chi.URLParam(r, "consumerGroupName")
	utils.Logger.Debug("render consumer group detail", "cluster", clusterName, "group", groupName)

	_, ok := s.clusterService.GetCluster(clusterName)
	if !ok {
		http.Error(w, "cluster not found", http.StatusNotFound)
		return
	}

	service := application.NewConsumerGroupsService(s.clusterService)
	group, err := service.FetchConsumerGroupWithLag(r.Context(), clusterName, groupName)
	if err != nil {
		utils.Logger.Error("fetch consumer group detail failed", "cluster", clusterName, "group", groupName, "err", err)
		http.Error(w, "failed to fetch consumer group detail", 500)
		return
	}

	if group.Group == "" {
		http.Error(w, "consumer group not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.ConsumerGroupDetail(clusterName, group).Render(r.Context(), w); err != nil {
		utils.Logger.Error("render consumer group detail view failed", "err", err)
		http.Error(w, "failed to render consumer group detail view", 500)
		return
	}
}
