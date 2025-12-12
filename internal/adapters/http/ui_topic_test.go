package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OliveiraNt/kdash/internal/config"
)

func TestUITopicsList(t *testing.T) {
	s := buildServer(t)

	// Test with non-existent cluster
	req := httptest.NewRequest(http.MethodGet, "/cluster/nonexistent/topics", nil)
	rec := httptest.NewRecorder()
	ctx := chiCtxWithParam("name", "nonexistent", req)
	s.uiTopicsList(rec, req.WithContext(ctx))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}

	// Add a cluster and test
	body, _ := json.Marshal(config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}})
	s.apiAddCluster(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/clusters", bytes.NewReader(body)))

	req = httptest.NewRequest(http.MethodGet, "/cluster/dev/topics", nil)
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParam("name", "dev", req)
	s.uiTopicsList(rec, req.WithContext(ctx))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

}

func TestUITopicDetail(t *testing.T) {
	s := buildServer(t)

	// Test with non-existent cluster
	req := httptest.NewRequest(http.MethodGet, "/cluster/nonexistent/topics/test-topic", nil)
	rec := httptest.NewRecorder()
	ctx := chiCtxWithParams(map[string]string{"name": "nonexistent", "topic": "test-topic"}, req)
	s.uiTopicDetail(rec, req.WithContext(ctx))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}

	// Add a cluster and test
	body, _ := json.Marshal(config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}})
	s.apiAddCluster(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/clusters", bytes.NewReader(body)))

	// Test successful topic detail
	req = httptest.NewRequest(http.MethodGet, "/cluster/dev/topics/test-topic", nil)
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParams(map[string]string{"name": "dev", "topic": "test-topic"}, req)
	s.uiTopicDetail(rec, req.WithContext(ctx))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("expected content-type text/html, got %s", rec.Header().Get("Content-Type"))
	}
}
