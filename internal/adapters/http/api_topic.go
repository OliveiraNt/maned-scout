package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/OliveiraNt/kdash/internal/domain"
	"github.com/OliveiraNt/kdash/internal/registry"

	"github.com/go-chi/chi/v5"
)

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
