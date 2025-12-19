package httpserver

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/OliveiraNt/maned-scout/internal/adapters/http/ui/templates/pages"
	"github.com/OliveiraNt/maned-scout/internal/application"
	"github.com/OliveiraNt/maned-scout/internal/domain"
	"github.com/OliveiraNt/maned-scout/internal/registry"

	"github.com/go-chi/chi/v5"
)

// mapErrorToHTTPStatus maps application errors to HTTP status codes
func mapErrorToHTTPStatus(err error) int {
	switch {
	case errors.Is(err, application.ErrClusterNotFound):
		return http.StatusNotFound
	case errors.Is(err, application.ErrInvalidTopicName),
		errors.Is(err, application.ErrInvalidPartitionCount),
		errors.Is(err, application.ErrInvalidReplicationFactor),
		errors.Is(err, application.ErrInvalidTopicConfig):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func (s *Server) apiListTopics(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "clusterName")
	showInternal := r.URL.Query().Get("showInternal") == "true"
	topics, err := s.topicService.ListTopics(name, showInternal)
	if err != nil {
		registry.Logger.Error("api list topics failed", "cluster", name, "err", err)
		w.Header().Set("X-Notification-Type", "error")
		w.Header().Set("X-Notification", "Falha ao listar tópicos")
		w.Header().Set("X-Notification-Base64", base64.StdEncoding.EncodeToString([]byte("Falha ao listar tópicos")))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.TopicsListFragment(name, topics, false).Render(r.Context(), w); err != nil {
		registry.Logger.Error("render topics list fragment failed", "cluster", name, "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	{
		totalTopics := len(topics)
		totalPartitions := 0
		for _, p := range topics {
			totalPartitions += p
		}
		avg := 0.0
		if totalTopics > 0 {
			avg = float64(totalPartitions) / float64(totalTopics)
		}
		_, _ = w.Write([]byte("<p class=\"text-3xl font-bold text-gray-900 dark:text-white mt-2\" id=\"topics-total\" hx-swap-oob=\"true\">" + strconv.Itoa(totalTopics) + "</p>"))
		_, _ = w.Write([]byte("<p class=\"text-3xl font-bold text-gray-900 dark:text-white mt-2\" id=\"partitions-total\" hx-swap-oob=\"true\">" + strconv.Itoa(totalPartitions) + "</p>"))
		_, _ = w.Write([]byte("<p class=\"text-3xl font-bold text-gray-900 dark:text-white mt-2\" id=\"partitions-avg\" hx-swap-oob=\"true\">" + strconv.FormatFloat(avg, 'f', 2, 64) + "</p>"))
	}
}

func (s *Server) apiGetTopicDetail(w http.ResponseWriter, r *http.Request) {
	clusterName := chi.URLParam(r, "clusterName")
	topicName := chi.URLParam(r, "topicName")

	topicDetail, err := s.topicService.GetTopicDetail(clusterName, topicName)
	if err != nil {
		registry.Logger.Error("api get topic detail failed", "cluster", clusterName, "topic", topicName, "err", err)
		http.Error(w, err.Error(), mapErrorToHTTPStatus(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(topicDetail); err != nil {
		registry.Logger.Error("encode topic detail failed", "cluster", clusterName, "topic", topicName, "err", err)
	}
}

func (s *Server) apiCreateTopic(w http.ResponseWriter, r *http.Request) {
	clusterName := chi.URLParam(r, "clusterName")
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
		_ = r.ParseForm()
		req.Name = strings.TrimSpace(r.FormValue("name"))
		if np, err := strconv.Atoi(r.FormValue("numPartitions")); err == nil {
			req.NumPartitions = int32(np)
		}
		if rf, err := strconv.Atoi(r.FormValue("replicationFactor")); err == nil {
			req.ReplicationFactor = int16(rf)
		}
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

	if err := s.topicService.CreateTopic(clusterName, req); err != nil {
		w.Header().Set("X-Notification-Type", "error")
		{
			msg := "Falha ao criar tópico: " + err.Error()
			w.Header().Set("X-Notification", msg)
			w.Header().Set("X-Notification-Base64", base64.StdEncoding.EncodeToString([]byte(msg)))
		}
		http.Error(w, err.Error(), mapErrorToHTTPStatus(err))
		return
	}

	w.Header().Set("X-Notification-Type", "success")
	{
		msg := "Tópico criado com sucesso"
		w.Header().Set("X-Notification", msg)
		w.Header().Set("X-Notification-Base64", base64.StdEncoding.EncodeToString([]byte(msg)))
	}

	w.Header().Set("HX-Trigger", "topic-created")
	w.WriteHeader(http.StatusNoContent)
	return
}

func (s *Server) apiDeleteTopic(w http.ResponseWriter, r *http.Request) {
	clusterName := chi.URLParam(r, "clusterName")
	topicName := chi.URLParam(r, "topicName")

	if err := s.topicService.DeleteTopic(clusterName, topicName); err != nil {
		registry.Logger.Error("api delete topic failed", "cluster", clusterName, "topic", topicName, "err", err)
		http.Error(w, err.Error(), mapErrorToHTTPStatus(err))
		return
	}

	w.WriteHeader(204)
}

func (s *Server) apiUpdateTopicConfig(w http.ResponseWriter, r *http.Request) {
	clusterName := chi.URLParam(r, "clusterName")
	topicName := chi.URLParam(r, "topicName")

	var req domain.UpdateTopicConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		registry.Logger.Warn("api update topic config bad request", "cluster", clusterName, "topic", topicName, "err", err)
		http.Error(w, err.Error(), 400)
		return
	}

	if err := s.topicService.UpdateTopicConfig(clusterName, topicName, req); err != nil {
		registry.Logger.Error("api update topic config failed", "cluster", clusterName, "topic", topicName, "err", err)
		http.Error(w, err.Error(), mapErrorToHTTPStatus(err))
		return
	}

	w.WriteHeader(200)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "topic config updated successfully"}); err != nil {
		registry.Logger.Error("encode response failed", "err", err)
	}
}

func (s *Server) apiIncreasePartitions(w http.ResponseWriter, r *http.Request) {
	clusterName := chi.URLParam(r, "clusterName")
	topicName := chi.URLParam(r, "topicName")

	var req domain.IncreasePartitionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		registry.Logger.Warn("api increase partitions bad request", "cluster", clusterName, "topic", topicName, "err", err)
		http.Error(w, err.Error(), 400)
		return
	}

	if err := s.topicService.IncreasePartitions(clusterName, topicName, req); err != nil {
		registry.Logger.Error("api increase partitions failed", "cluster", clusterName, "topic", topicName, "err", err)
		http.Error(w, err.Error(), mapErrorToHTTPStatus(err))
		return
	}

	w.WriteHeader(200)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "partitions increased successfully"}); err != nil {
		registry.Logger.Error("encode response failed", "err", err)
	}
}

func (s *Server) apiReadMessages(w http.ResponseWriter, r *http.Request) {
	clusterName := chi.URLParam(r, "clusterName")
	topicName := chi.URLParam(r, "topicName")
	if err := pages.MessageView(clusterName, topicName).Render(r.Context(), w); err != nil {
		registry.Logger.Error("render message view failed", "err", err)
		http.Error(w, "failed to render message view", 500)
		return
	}
}

func (s *Server) apiStopMessages(w http.ResponseWriter, r *http.Request) {
	clusterName := chi.URLParam(r, "clusterName")
	topicName := chi.URLParam(r, "topicName")
	if err := pages.StopView(clusterName, topicName).Render(r.Context(), w); err != nil {
		registry.Logger.Error("render stop view failed", "err", err)
		http.Error(w, "failed to render stop view", 500)
		return
	}
}

func (s *Server) apiWriteMessage(w http.ResponseWriter, r *http.Request) {
	clusterName := chi.URLParam(r, "clusterName")
	topicName := chi.URLParam(r, "topicName")

	var req domain.MessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		registry.Logger.Warn("api write message bad request", "cluster", clusterName, "topic", topicName, "err", err)
		http.Error(w, err.Error(), 400)
		return
	}

	m := domain.Message{
		Key:   []byte(req.Key),
		Value: []byte(req.Value),
	}
	if err := s.topicService.WriteMessage(clusterName, topicName, m); err != nil {
		registry.Logger.Error("api write message failed", "cluster", clusterName, "topic", topicName, "err", err)
		http.Error(w, err.Error(), mapErrorToHTTPStatus(err))
		return
	}
}
