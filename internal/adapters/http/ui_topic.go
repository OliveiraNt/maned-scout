package httpserver

import (
	"net/http"

	"github.com/OliveiraNt/kdash/internal/adapters/http/ui/templates/pages"
	"github.com/OliveiraNt/kdash/internal/domain"
	"github.com/OliveiraNt/kdash/internal/registry"

	"github.com/go-chi/chi/v5"
)

func (s *Server) uiTopicsList(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	registry.Logger.Debug("render topics list", "cluster", name)
	_, ok := s.clusterService.GetCluster(name)
	if !ok {
		http.Error(w, "cluster not found", http.StatusNotFound)
		return
	}
	topics := make(map[string]int)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.TopicsList(name, topics).Render(r.Context(), w); err != nil {
		registry.Logger.Error("render topics list failed", "cluster", name, "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) uiTopicDetail(w http.ResponseWriter, r *http.Request) {
	clusterName := chi.URLParam(r, "name")
	topicName := chi.URLParam(r, "topic")
	registry.Logger.Debug("render topic detail", "cluster", clusterName, "topic", topicName)
	_, ok := s.clusterService.GetCluster(clusterName)
	if !ok {
		http.Error(w, "cluster not found", http.StatusNotFound)
		return
	}
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
