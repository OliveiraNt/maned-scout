package httpserver

import (
	"net/http"

	"github.com/OliveiraNt/maned-scout/internal/adapters/http/ui/templates/pages"
	"github.com/OliveiraNt/maned-scout/internal/utils"

	"github.com/go-chi/chi/v5"
)

func (s *Server) uiTopicsList(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "clusterName")
	utils.Logger.Debug("render topics list", "cluster", name)
	_, ok := s.clusterService.GetCluster(name)
	if !ok {
		http.Error(w, "cluster not found", http.StatusNotFound)
		return
	}
	topics := make(map[string]int)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.TopicsList(name, topics).Render(r.Context(), w); err != nil {
		utils.Logger.Error("render topics list failed", "cluster", name, "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) uiTopicDetail(w http.ResponseWriter, r *http.Request) {
	clusterName := chi.URLParam(r, "clusterName")
	topicName := chi.URLParam(r, "topicName")
	utils.Logger.Debug("render topic detail", "cluster", clusterName, "topic", topicName)

	_, ok := s.clusterService.GetCluster(clusterName)
	if !ok {
		http.Error(w, "cluster not found", http.StatusNotFound)
		return
	}

	topicDetail, err := s.topicService.GetTopicDetail(clusterName, topicName)
	if err != nil {
		utils.Logger.Error("get topic detail failed", "cluster", clusterName, "topic", topicName, "err", err)
		http.Error(w, "topic not found", http.StatusNotFound)
		return
	}

	if topicDetail == nil {
		http.Error(w, "topic not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.TopicDetail(clusterName, topicDetail).Render(r.Context(), w); err != nil {
		utils.Logger.Error("render topic detail failed", "cluster", clusterName, "topic", topicName, "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
