package httpserver

import (
	"net/http"

	"github.com/OliveiraNt/kdash/internal/adapters/http/ui/templates/pages"
	"github.com/OliveiraNt/kdash/internal/domain"
	"github.com/OliveiraNt/kdash/internal/registry"

	"github.com/go-chi/chi/v5"
)

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
