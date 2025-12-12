package httpserver

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	pages "github.com/OliveiraNt/kdash/internal/adapters/http/ui/templates/pages"
	"github.com/OliveiraNt/kdash/internal/domain"
	"github.com/OliveiraNt/kdash/internal/registry"

	"github.com/go-chi/chi/v5"
)

func (s *Server) apiListTopics(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	client, ok := s.repo.GetClient(name)
	if !ok {
		registry.Logger.Warn("api list topics cluster not found", "cluster", name)
		w.Header().Set("X-Notification-Type", "error")
		w.Header().Set("X-Notification", "Cluster não encontrado")
		w.Header().Set("X-Notification-Base64", base64.StdEncoding.EncodeToString([]byte("Cluster não encontrado")))
		http.Error(w, "cluster not found", http.StatusNotFound)
		return
	}
	// Preserve current filter
	showInternal := r.URL.Query().Get("showInternal") == "true"
	topics, err := client.ListTopics(showInternal)
	if err != nil {
		registry.Logger.Error("api list topics failed", "cluster", name, "err", err)
		w.Header().Set("X-Notification-Type", "error")
		w.Header().Set("X-Notification", "Falha ao listar tópicos")
		w.Header().Set("X-Notification-Base64", base64.StdEncoding.EncodeToString([]byte("Falha ao listar tópicos")))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.TopicsListFragment(name, topics, showInternal, true).Render(r.Context(), w); err != nil {
		registry.Logger.Error("render topics list fragment failed", "cluster", name, "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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
	// Determine if request is JSON or form (htmx form submission)
	var req domain.CreateTopicRequest
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			registry.Logger.Warn("api create topic bad request", "cluster", clusterName, "err", err)
			w.Header().Set("X-Notification-Type", "error")
			{
				msg := "Requisição inválida: " + err.Error()
				w.Header().Set("X-Notification", msg)
				w.Header().Set("X-Notification-Base64", base64.StdEncoding.EncodeToString([]byte(msg)))
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		// Parse form values
		_ = r.ParseForm()
		req.Name = strings.TrimSpace(r.FormValue("name"))
		if np, err := strconv.Atoi(r.FormValue("numPartitions")); err == nil {
			req.NumPartitions = int32(np)
		}
		if rf, err := strconv.Atoi(r.FormValue("replicationFactor")); err == nil {
			req.ReplicationFactor = int16(rf)
		}
		// Optional configs (pass-through as strings)
		cfgs := map[string]*string{}
		if v := strings.TrimSpace(r.FormValue("retention.ms")); v != "" {
			cfgs["retention.ms"] = &v
		}
		if v := strings.TrimSpace(r.FormValue("cleanup.policy")); v != "" {
			cfgs["cleanup.policy"] = &v
		}
		if v := strings.TrimSpace(r.FormValue("compression.type")); v != "" {
			cfgs["compression.type"] = &v
		}
		if len(cfgs) > 0 {
			req.Configs = cfgs
		}
	}

	// Validate request
	if req.Name == "" {
		w.Header().Set("X-Notification-Type", "error")
		{
			msg := "Nome do tópico é obrigatório"
			w.Header().Set("X-Notification", msg)
			w.Header().Set("X-Notification-Base64", base64.StdEncoding.EncodeToString([]byte(msg)))
		}
		http.Error(w, "topic name is required", http.StatusBadRequest)
		return
	}
	if req.NumPartitions <= 0 {
		w.Header().Set("X-Notification-Type", "error")
		{
			msg := "Número de partições deve ser maior que 0"
			w.Header().Set("X-Notification", msg)
			w.Header().Set("X-Notification-Base64", base64.StdEncoding.EncodeToString([]byte(msg)))
		}
		http.Error(w, "number of partitions must be greater than 0", http.StatusBadRequest)
		return
	}
	if req.ReplicationFactor <= 0 {
		w.Header().Set("X-Notification-Type", "error")
		{
			msg := "Fator de replicação deve ser maior que 0"
			w.Header().Set("X-Notification", msg)
			w.Header().Set("X-Notification-Base64", base64.StdEncoding.EncodeToString([]byte(msg)))
		}
		http.Error(w, "replication factor must be greater than 0", http.StatusBadRequest)
		return
	}

	client, ok := s.repo.GetClient(clusterName)
	if !ok {
		registry.Logger.Warn("api create topic cluster not found", "cluster", clusterName)
		w.Header().Set("X-Notification-Type", "error")
		{
			msg := "Cluster não encontrado"
			w.Header().Set("X-Notification", msg)
			w.Header().Set("X-Notification-Base64", base64.StdEncoding.EncodeToString([]byte(msg)))
		}
		http.Error(w, "cluster not found", http.StatusNotFound)
		return
	}

	if err := client.CreateTopic(req); err != nil {
		registry.Logger.Error("api create topic failed", "cluster", clusterName, "topic", req.Name, "err", err)
		w.Header().Set("X-Notification-Type", "error")
		{
			msg := "Falha ao criar tópico: " + err.Error()
			w.Header().Set("X-Notification", msg)
			w.Header().Set("X-Notification-Base64", base64.StdEncoding.EncodeToString([]byte(msg)))
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// On success, return updated list fragment (HTML) suitable for htmx swap
	registry.Logger.Info("topic created", "cluster", clusterName, "topic", req.Name)

	// Check if caller wants to include internal topics
	showInternal := false
	// from form
	if r.FormValue("showInternal") == "true" {
		showInternal = true
	}
	// or query param (JSON callers)
	if r.URL.Query().Get("showInternal") == "true" {
		showInternal = true
	}

	topics, err := client.ListTopics(showInternal)
	if err != nil {
		registry.Logger.Error("list topics after create failed", "cluster", clusterName, "err", err)
		w.Header().Set("X-Notification-Type", "error")
		{
			msg := "Tópico criado, mas falhou ao atualizar a lista"
			w.Header().Set("X-Notification", msg)
			w.Header().Set("X-Notification-Base64", base64.StdEncoding.EncodeToString([]byte(msg)))
		}
		// Don't attempt to render list; return 200 so frontend keeps working
		w.WriteHeader(http.StatusOK)
		return
	}

	// Success: render fragment and notify
	w.Header().Set("X-Notification-Type", "success")
	{
		msg := "Tópico criado com sucesso"
		w.Header().Set("X-Notification", msg)
		w.Header().Set("X-Notification-Base64", base64.StdEncoding.EncodeToString([]byte(msg)))
	}
	// Ask HTMX to trigger a custom event so the client can close the modal & reset the form
	// Use After-Swap to ensure OOB updates are applied before closing the modal
	w.Header().Set("HX-Trigger-After-Swap", "topic-created")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.TopicsListFragment(clusterName, topics, showInternal, true).Render(r.Context(), w); err != nil {
		registry.Logger.Error("render topics list fragment after create failed", "cluster", clusterName, "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// removed HTML string builders in favor of templ components

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
