package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/OliveiraNt/kdash/internal/config"
	"github.com/OliveiraNt/kdash/internal/domain"
)

func TestAPIListTopics(t *testing.T) {
	// Build server with pre-configured topics for the "dev" cluster
	factory := testFactory{
		topicsPerCluster: map[string]map[string]int{
			"dev": {"a": 1, "b": 3},
		},
	}
	s := buildServerWithFactory(t, factory)

	// Add a cluster via API
	body, _ := json.Marshal(config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}})
	s.apiAddCluster(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/clusters", bytes.NewReader(body)))

	req := httptest.NewRequest(http.MethodGet, "/api/clusters/dev/topics", nil)
	rec := httptest.NewRecorder()
	ctx := chiCtxWithParam("clusterName", "dev", req)
	s.apiListTopics(rec, req.WithContext(ctx))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	// Now returns HTML fragment for HTMX; assert content type and that topic names appear
	if ct := rec.Header().Get("Content-Type"); ct == "" || ct[:9] != "text/html" {
		t.Fatalf("expected text/html content type, got %q", rec.Header().Get("Content-Type"))
	}
	htmlBody := rec.Body.String()
	if !strings.Contains(htmlBody, ">a<") || !strings.Contains(htmlBody, ">b<") {
		t.Fatalf("expected HTML to contain topic names, got: %s", htmlBody)
	}
}

func TestAPIGetTopicDetail(t *testing.T) {
	s := buildServer(t)
	// Add a cluster via API
	body, _ := json.Marshal(config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}})
	s.apiAddCluster(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/clusters", bytes.NewReader(body)))

	// Test successful topic detail retrieval
	req := httptest.NewRequest(http.MethodGet, "/api/clusters/dev/topics/test-topic", nil)
	rec := httptest.NewRecorder()
	ctx := chiCtxWithParams(map[string]string{"clusterName": "dev", "topicName": "test-topic"}, req)
	s.apiGetTopicDetail(rec, req.WithContext(ctx))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Test with non-existent cluster
	req = httptest.NewRequest(http.MethodGet, "/api/clusters/nonexistent/topics/test-topic", nil)
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParams(map[string]string{"clusterName": "nonexistent", "topicName": "test-topic"}, req)
	s.apiGetTopicDetail(rec, req.WithContext(ctx))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestAPICreateTopic(t *testing.T) {
	s := buildServer(t)
	// Add a cluster via API
	body, _ := json.Marshal(config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}})
	s.apiAddCluster(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/clusters", bytes.NewReader(body)))

	// Test successful topic creation
	createReq := domain.CreateTopicRequest{
		Name:              "new-topic",
		NumPartitions:     3,
		ReplicationFactor: 1,
	}
	body, _ = json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/clusters/dev/topics", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	ctx := chiCtxWithParam("clusterName", "dev", req)
	s.apiCreateTopic(rec, req.WithContext(ctx))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	// Should trigger topic-created event for HTMX clients
	if trig := rec.Result().Header.Get("HX-Trigger"); !strings.Contains(trig, "topic-created") {
		t.Fatalf("expected HX-Trigger to contain 'topic-created', got %q", trig)
	}

	// Test with missing topic name
	badReq := domain.CreateTopicRequest{
		Name:              "",
		NumPartitions:     3,
		ReplicationFactor: 1,
	}
	body, _ = json.Marshal(badReq)
	req = httptest.NewRequest(http.MethodPost, "/api/clusters/dev/topics", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParam("clusterName", "dev", req)
	s.apiCreateTopic(rec, req.WithContext(ctx))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty name, got %d", rec.Code)
	}

	// Test with invalid partitions
	badReq = domain.CreateTopicRequest{
		Name:              "topic",
		NumPartitions:     0,
		ReplicationFactor: 1,
	}
	body, _ = json.Marshal(badReq)
	req = httptest.NewRequest(http.MethodPost, "/api/clusters/dev/topics", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParam("name", "dev", req)
	s.apiCreateTopic(rec, req.WithContext(ctx))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for zero partitions, got %d", rec.Code)
	}

	// Test with invalid replication factor
	badReq = domain.CreateTopicRequest{
		Name:              "topic",
		NumPartitions:     3,
		ReplicationFactor: 0,
	}
	body, _ = json.Marshal(badReq)
	req = httptest.NewRequest(http.MethodPost, "/api/clusters/dev/topics", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParam("clusterName", "dev", req)
	s.apiCreateTopic(rec, req.WithContext(ctx))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for zero replication factor, got %d", rec.Code)
	}

	// Test with non-existent cluster
	body, _ = json.Marshal(createReq)
	req = httptest.NewRequest(http.MethodPost, "/api/clusters/nonexistent/topics", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParam("clusterName", "nonexistent", req)
	s.apiCreateTopic(rec, req.WithContext(ctx))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestAPIDeleteTopic(t *testing.T) {
	s := buildServer(t)
	// Add a cluster via API
	body, _ := json.Marshal(config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}})
	s.apiAddCluster(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/clusters", bytes.NewReader(body)))

	// Test successful topic deletion
	req := httptest.NewRequest(http.MethodDelete, "/api/clusters/dev/topics/test-topic", nil)
	rec := httptest.NewRecorder()
	ctx := chiCtxWithParams(map[string]string{"clusterName": "dev", "topicName": "test-topic"}, req)
	s.apiDeleteTopic(rec, req.WithContext(ctx))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}

	// Test with non-existent cluster
	req = httptest.NewRequest(http.MethodDelete, "/api/clusters/nonexistent/topics/test-topic", nil)
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParams(map[string]string{"clusterName": "nonexistent", "topicName": "test-topic"}, req)
	s.apiDeleteTopic(rec, req.WithContext(ctx))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestAPIUpdateTopicConfig(t *testing.T) {
	s := buildServer(t)
	// Add a cluster via API
	body, _ := json.Marshal(config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}})
	s.apiAddCluster(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/clusters", bytes.NewReader(body)))

	// Test successful config update
	val := "1000"
	updateReq := domain.UpdateTopicConfigRequest{
		Configs: map[string]*string{
			"retention.ms": &val,
		},
	}
	body, _ = json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/api/clusters/dev/topics/test-topic/config", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	ctx := chiCtxWithParams(map[string]string{"clusterName": "dev", "topicName": "test-topic"}, req)
	s.apiUpdateTopicConfig(rec, req.WithContext(ctx))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Test with empty configs
	emptyReq := domain.UpdateTopicConfigRequest{
		Configs: map[string]*string{},
	}
	body, _ = json.Marshal(emptyReq)
	req = httptest.NewRequest(http.MethodPut, "/api/clusters/dev/topics/test-topic/config", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParams(map[string]string{"clusterName": "dev", "topicName": "test-topic"}, req)
	s.apiUpdateTopicConfig(rec, req.WithContext(ctx))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty configs, got %d", rec.Code)
	}

	// Test with non-existent cluster
	body, _ = json.Marshal(updateReq)
	req = httptest.NewRequest(http.MethodPut, "/api/clusters/nonexistent/topics/test-topic/config", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParams(map[string]string{"clusterName": "nonexistent", "topicName": "test-topic"}, req)
	s.apiUpdateTopicConfig(rec, req.WithContext(ctx))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestAPIIncreasePartitions(t *testing.T) {
	s := buildServer(t)
	// Add a cluster via API
	body, _ := json.Marshal(config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}})
	s.apiAddCluster(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/clusters", bytes.NewReader(body)))

	// Test successful partition increase
	increaseReq := domain.IncreasePartitionsRequest{
		TotalPartitions: 5,
	}
	body, _ = json.Marshal(increaseReq)
	req := httptest.NewRequest(http.MethodPost, "/api/clusters/dev/topics/test-topic/partitions", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	ctx := chiCtxWithParams(map[string]string{"clusterName": "dev", "topicName": "test-topic"}, req)
	s.apiIncreasePartitions(rec, req.WithContext(ctx))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Test with invalid partitions
	badReq := domain.IncreasePartitionsRequest{
		TotalPartitions: 0,
	}
	body, _ = json.Marshal(badReq)
	req = httptest.NewRequest(http.MethodPost, "/api/clusters/dev/topics/test-topic/partitions", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParams(map[string]string{"clusterName": "dev", "topicName": "test-topic"}, req)
	s.apiIncreasePartitions(rec, req.WithContext(ctx))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for zero partitions, got %d", rec.Code)
	}

	// Test with non-existent cluster
	body, _ = json.Marshal(increaseReq)
	req = httptest.NewRequest(http.MethodPost, "/api/clusters/nonexistent/topics/test-topic/partitions", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParams(map[string]string{"clusterName": "nonexistent", "topicName": "test-topic"}, req)
	s.apiIncreasePartitions(rec, req.WithContext(ctx))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
